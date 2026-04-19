## Why

The proxy has no rate limiting, so clients using OpenAI-compatible tooling cannot rely on standard OpenAI rate limit headers for backoff or quota tracking. Adding OpenAI-compatible rate limiting makes the proxy a drop-in replacement for production workflows that depend on `x-ratelimit-*` headers and proper 429 responses.

## What Changes

- Add configurable rate limiting middleware (requests-per-minute and tokens-per-minute) that enforces limits per API key or globally.
- Return OpenAI-compatible `x-ratelimit-*` response headers on every `/v1/chat/completions` and `/v1/models` request.
- Return HTTP 429 with an OpenAI-compatible JSON error body and `Retry-After` header when a limit is exceeded.
- Extend the config file schema with a `rate_limit` section.

## Capabilities

### New Capabilities

- `rate-limiting`: OpenAI-compatible HTTP rate limiting middleware — enforces RPM and TPM limits, returns `x-ratelimit-*` headers and 429 errors with `retry-after`.

### Modified Capabilities

- `server-config`: Add `rate_limit` block to the YAML config schema (new fields, no existing fields removed — non-breaking).

## Impact

- `internal/proxy/handler.go` — wrap handlers with rate limit middleware.
- `internal/config/config.go` — add `RateLimit` struct to `Config`.
- New package `internal/ratelimit/` — token bucket / sliding-window counters, middleware.
- Config file format gains optional `rate_limit` section (backwards-compatible).
- No external dependencies required (stdlib `sync` + `time` suffice).