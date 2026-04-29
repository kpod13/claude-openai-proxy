## 1. Startup Model Names

- [x] 1.1 Update `cmd/server/main.go` "Models discovered" log call to include a `models` attribute listing all model IDs (extracted from `reg.List()`)

## 2. Logger on Handler

- [x] 2.1 Add `Logger *slog.Logger` field to `proxy.Handler` in `internal/proxy/handler.go`
- [x] 2.2 Update `cmd/server/main.go` to set `Handler.Logger` when constructing the handler

## 3. HTTP Request/Response Debug Logging

- [x] 3.1 In `Handler.ChatCompletions`, log the incoming request (method, path, model) at DEBUG level, guarded by nil-logger check
- [x] 3.2 In `handleBlocking` and `handleStreaming`, log the response outcome (status, duration) at DEBUG level

## 4. CLI Invocation Debug Logging

- [x] 4.1 In `Handler.handleBlocking`, log before the CLI call (model, prompt char length) and after (input tokens, output tokens, duration) at DEBUG level
- [x] 4.2 In `Handler.handleStreaming`, log before the CLI call (model, prompt char length) and after stream start at DEBUG level

## 5. Tests

- [x] 5.1 Add test for startup log including model names (verify `models` attribute appears)
- [x] 5.2 Add test that `Handler` with a DEBUG-level logger emits request and CLI debug entries on a chat completion call
- [x] 5.3 Add test that `Handler` with a nil logger handles requests without panicking
