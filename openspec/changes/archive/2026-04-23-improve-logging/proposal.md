## Why

The proxy's logging is incomplete in two ways: startup messages omit useful detail (model names are not printed), and verbose mode (`--verbose`) produces no debug output because the logger is never wired into the HTTP handler or CLI runner. Operators have no visibility into request flow even when they opt into verbose logging.

## What Changes

- Log discovered model names (not just count) in the "Models discovered" startup message.
- Add a `Logger` field to `proxy.Handler` so HTTP handlers can emit debug logs.
- Log incoming HTTP request (method, path, model) and outgoing HTTP response (status, duration) at DEBUG level.
- Log CLI invocations (model, prompt length) and results (tokens, latency) at DEBUG level in `RunBlocking` and `RunStreaming`.
- Wire the logger from `main.go` into `proxy.Handler` and the CLI runner functions.

## Capabilities

### New Capabilities

*(none)*

### Modified Capabilities

- `logging`: Add requirements for verbose (DEBUG-level) HTTP request/response logging and CLI invocation logging, and for model names appearing in the startup discovery message.

## Impact

- `cmd/server/main.go`: pass logger to `proxy.Handler`; update "Models discovered" log call.
- `internal/proxy/handler.go`: add `Logger *slog.Logger` field; add debug logging in `ChatCompletions`, `handleBlocking`, `handleStreaming`.
- `internal/proxy/claude.go`: thread logger into `RunBlocking` / `RunStreaming` (via context value or explicit parameter).
- `internal/proxy/models.go`: add `IDs()` or similar helper to expose model name list for the startup log.
- No external API changes; no new dependencies.
