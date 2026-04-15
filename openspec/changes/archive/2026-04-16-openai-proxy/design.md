## Context

The server currently has a single `/healthz` endpoint. The OpenAI chat-completions API surface we need to emulate is small: list models and handle chat completions (blocking + streaming). The `claude` CLI is the only interface to Claude available in this environment — there is no direct API access. All HTTP-to-Claude translation must go through subprocess invocations.

## Goals / Non-Goals

**Goals:**
- `GET /v1/models` returns a valid OpenAI-format model list populated from live CLI probes
- `POST /v1/chat/completions` handles both `stream: false` (JSON response) and `stream: true` (SSE)
- Multi-turn conversations are supported by serializing the messages array into a prompt
- No external Go dependencies beyond the standard library

**Non-Goals:**
- Embeddings, images, audio, function-calling, tool use
- Authentication / API key validation
- Rate limiting, retries, or load balancing
- Persistent conversation sessions (each request is stateless)

## Decisions

### Model discovery at startup
Run `claude --print --output-format json --model <alias> "."` for each known alias (`opus`, `sonnet`, `haiku`) at server startup. Parse the `modelUsage` map in the JSON result to get the resolved full model ID. Cache the result in memory. This avoids hard-coding version suffixes that change with each Claude release.

**Alternative considered**: hard-coded list — rejected because model IDs include version suffixes (`claude-sonnet-4-6`) that will change and need manual maintenance.

**Alternative considered**: reading `~/.claude/settings.json` — contains no model list.

### Message serialization
Convert the OpenAI `messages` array into a single prompt string passed to `claude --print`:
- `system` role → prepended as `[System]: <content>\n`
- `user` role → `[User]: <content>\n`
- `assistant` role → `[Assistant]: <content>\n`
The final line is always a `user` message; the assembled string is passed via stdin with `--input-format text`.

**Alternative considered**: `--system-prompt` flag for the system message + only last user turn as prompt — simpler but loses conversation history, breaking multi-turn use cases.

### Streaming (SSE)
For `stream: true`, invoke `claude --print --output-format stream-json --verbose --model <model>` and parse each newline-delimited JSON object from stdout:
- `type: "assistant"` objects contain partial `content[].text` chunks → emit as `data: <openai-chunk>\n\n` SSE events
- `type: "result"` → emit `data: [DONE]\n\n` and close the stream
Set `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `X-Accel-Buffering: no`.

**Alternative considered**: buffer full response then fake streaming — rejected because it defeats the purpose and increases time-to-first-token.

### Package layout
```
internal/
  proxy/
    handler.go     — HTTP handlers (models, chat completions)
    claude.go      — subprocess invocation + output parsing
    models.go      — model discovery and registry
    types.go       — OpenAI-compatible request/response structs
```
`cmd/server/main.go` imports `internal/proxy` and registers routes.

### Model name pass-through
If the request's `model` field matches a discovered full Claude ID (`claude-sonnet-4-6`), use it directly. If it matches an alias (`sonnet`, `opus`, `haiku`), resolve to the full ID. If unknown, return HTTP 400.

## Risks / Trade-offs

- **Startup latency** (3 subprocess probes) → probes run concurrently with `sync.WaitGroup`; total startup overhead ≈ one round-trip
- **claude CLI not on PATH** → server fails fast at startup with a clear error message
- **Message format mismatch** → serialization covers the common cases; edge cases (image content, tool messages) return HTTP 400 with a descriptive error
- **CLI output format changes** → the `stream-json` and `json` formats are established; a version guard can be added if breakage is detected
- **Concurrent requests** → each request spawns its own subprocess; no shared state beyond the read-only model registry

## Migration Plan

Additive change only. The `/healthz` route is unchanged. Deploy by restarting the server binary.
