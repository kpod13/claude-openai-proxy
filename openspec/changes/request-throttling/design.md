## Context

The proxy currently forwards all requests to Claude CLI without any rate limiting. OpenAI clients (LangChain, openai-python, etc.) are built to handle `x-ratelimit-*` headers and 429 responses; without them, clients cannot implement proper backoff or quota tracking. The proxy is single-process Go, with no external state store, so any rate limiter must be in-process.

## Goals / Non-Goals

**Goals:**
- Return `x-ratelimit-*` headers on every `/v1/chat/completions` call, compatible with OpenAI's header schema.
- Enforce configurable RPM (requests per minute) and TPM (tokens per minute) limits.
- Return HTTP 429 with an OpenAI-compatible JSON error body and `Retry-After` header when a limit is breached.
- Support per-API-key limiting (keyed on `Authorization: Bearer <key>`) with a global fallback when no key is present.
- Config-driven: limits defined in the YAML config file; rate limiting is disabled by default (zero values = unlimited).

**Non-Goals:**
- Persistent quota storage across server restarts (in-memory only).
- Distributed rate limiting across multiple proxy instances.
- Per-model or per-endpoint limits (all limits are global or per-key).
- Rate limiting the `/v1/models` endpoint (read-only, cheap).

## Decisions

### Token bucket vs. sliding window

**Decision**: Use a fixed-window counter per 1-minute bucket.

**Rationale**: OpenAI uses fixed 1-minute windows for RPM and TPM (the reset headers are always ≤ 60 s). A fixed window is O(1) memory per key, trivial to implement without dependencies, and matches what clients expect from the `x-ratelimit-reset-*` headers (a countdown to the next minute boundary). A true sliding window would be more accurate but the added complexity is not worth it for a local proxy.

**Alternative considered**: Token bucket — more burst-friendly, but `x-ratelimit-remaining-*` and `x-ratelimit-reset-*` semantics map more naturally onto fixed windows.

### Per-key vs. global limiting

**Decision**: Rate limit keyed on the `Authorization` bearer token. If no token is present, all unauthenticated requests share a single global bucket.

**Rationale**: OpenAI limits are per-API-key. Mirroring this lets multi-tenant setups assign different keys to different callers.

### Middleware placement

**Decision**: Implement as an `http.Handler` middleware wrapping `ChatCompletions` only.

**Rationale**: `/v1/models` is a cheap in-memory list; applying rate limits there would break health-checks and model discovery in clients without meaningful benefit.

### Token counting

**Decision**: For pre-request RPM enforcement, tokens in the current request body are estimated (counted at request time). TPM decrement happens at request time using the estimated prompt token count; actual output tokens are not retroactively accounted for.

**Rationale**: The proxy cannot know output token count before Claude responds. Counting only prompt tokens (known at request time) is how most proxy implementations work and is acceptable for a local tool. If the response triggers a TPM 429, it returns after the fact — the same behavior as OpenAI when token estimates are off.

### Package structure

**Decision**: New `internal/ratelimit/` package. Exposes:
- `Limiter` struct with `Allow(key string, estimatedTokens int) (Info, bool)` method.
- `Middleware(l *Limiter) func(http.Handler) http.Handler`.
- `Info` carries the values needed to populate `x-ratelimit-*` headers.

**Rationale**: Keeps rate limit logic isolated and independently testable.

## Risks / Trade-offs

- **Token over-counting**: Prompt tokens are estimated by splitting on whitespace (cheap proxy for tiktoken). Actual Claude token counts differ. → Acceptable inaccuracy for a local proxy; users can tune limits.
- **In-memory state lost on restart**: Limits reset when the server restarts. → By design; documented in config comments.
- **Clock skew**: Fixed windows use `time.Now()` truncated to the minute. Rapid restart at a minute boundary could double the effective limit for one window. → Negligible for a local proxy.

## Migration Plan

1. Add `rate_limit` block to config (optional, default disabled).
2. New `internal/ratelimit/` package with `Limiter` and `Middleware`.
3. Wire middleware in `main.go` after config load.
4. All existing behaviour unchanged when `rate_limit` is absent from config.
5. No data migration needed (in-memory only).