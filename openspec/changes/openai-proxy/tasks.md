## 1. Package scaffolding

- [x] 1.1 Create `internal/proxy/` directory
- [x] 1.2 Create `internal/proxy/types.go` — OpenAI-compatible request/response structs (`ChatRequest`, `ChatResponse`, `ChatCompletionChunk`, `ModelList`, `ModelObject`, `Message`, `Usage`, `Choice`, `Delta`)

## 2. Model discovery

- [x] 2.1 Create `internal/proxy/models.go` with a `Registry` struct holding discovered models
- [x] 2.2 Implement `Discover(aliases []string) *Registry` — runs concurrent `claude --print --output-format json --model <alias> "."` probes, parses `modelUsage` keys to resolve full IDs
- [x] 2.3 Implement `Registry.Resolve(name string) (string, error)` — returns full model ID for a given full ID or alias, error if unknown

## 3. Claude CLI invocation

- [x] 3.1 Create `internal/proxy/claude.go`
- [x] 3.2 Implement `RunBlocking(model, prompt string) (*CLIResult, error)` — runs `claude --print --output-format json --model <model>`, passes prompt via stdin, parses JSON output into `CLIResult` (result text + usage)
- [x] 3.3 Implement `RunStreaming(ctx context.Context, model, prompt string) (<-chan StreamChunk, error)` — runs `claude --print --output-format stream-json --verbose --model <model>`, pipes stdout through a goroutine that emits `StreamChunk` values for each `type:"assistant"` JSON line, closes channel on `type:"result"`

## 4. HTTP handlers

- [x] 4.1 Create `internal/proxy/handler.go` with a `Handler` struct holding a `*Registry`
- [x] 4.2 Implement `Handler.Models(w, r)` — serializes registry contents as OpenAI `ModelList` JSON
- [x] 4.3 Implement `Handler.ChatCompletions(w, r)` — decodes `ChatRequest`, serializes messages to prompt, resolves model, dispatches to blocking or streaming path
- [x] 4.4 Implement blocking response path: call `RunBlocking`, map result to `ChatResponse`, write JSON
- [x] 4.5 Implement streaming response path: set SSE headers, call `RunStreaming`, range over chunks emitting `ChatCompletionChunk` SSE events, finish with `data: [DONE]`

## 5. Wire into main

- [x] 5.1 In `cmd/server/main.go`, call `proxy.Discover([]string{"opus", "sonnet", "haiku"})` at startup (fatal on zero models)
- [x] 5.2 Register `GET /v1/models` and `POST /v1/chat/completions` routes via the `Handler`

## 7. Unit tests

- [x] 7.1 `internal/proxy/models_test.go` — test `Registry.Resolve`: full ID pass-through, alias resolution, unknown model error
- [x] 7.2 `internal/proxy/handler_test.go` — test `serializeMessages`: system/user/assistant formatting, unsupported role error
- [x] 7.3 `internal/proxy/handler_test.go` — test `Handler.Models` via `httptest`: correct JSON shape and model IDs
- [x] 7.4 `internal/proxy/handler_test.go` — test `Handler.ChatCompletions` bad request (malformed JSON → 400)
- [x] 7.5 `internal/proxy/handler_test.go` — test `Handler.ChatCompletions` unknown model → 400
- [x] 7.6 `internal/proxy/claude_test.go` — test `parseBlockingOutput`: valid JSON → CLIResult, missing JSON → error
- [x] 7.7 Run `go test ./...` — all tests pass

## 6. Verification

- [x] 6.1 Run `go build ./...` — must succeed with no errors
- [x] 6.2 Start the server and call `GET /v1/models` — verify JSON response contains at least one model
- [x] 6.3 Call `POST /v1/chat/completions` with `stream: false` — verify valid OpenAI-format JSON response
- [x] 6.4 Call `POST /v1/chat/completions` with `stream: true` — verify SSE stream with `data: [DONE]` terminator
