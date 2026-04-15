## Why

Tools and IDEs that speak the OpenAI chat-completions protocol (Cursor, Continue, LiteLLM, etc.) cannot directly use the `claude` CLI. A thin HTTP proxy that translates OpenAI API calls into `claude --print` invocations lets any OpenAI-compatible client use Claude without API key management or SDK changes.

## What Changes

- Add an HTTP server handler that implements the OpenAI-compatible endpoints:
  - `GET /v1/models` — returns available Claude models discovered from the CLI
  - `POST /v1/chat/completions` — translates OpenAI chat messages and returns Claude's response in OpenAI format, with optional SSE streaming
- Add model discovery: at startup the server probes the `claude` CLI with known model aliases (`opus`, `sonnet`, `haiku`) to resolve their full model IDs and build the `/v1/models` list
- Wire the new handlers into the existing `cmd/server/main.go` entry point

## Capabilities

### New Capabilities

- `model-discovery`: Probing the `claude` CLI at startup to resolve available model IDs
- `chat-completions`: Translating OpenAI `/v1/chat/completions` requests into `claude --print` invocations and mapping the response back to OpenAI format (both blocking and SSE streaming)
- `models-endpoint`: Serving `GET /v1/models` with the discovered model list in OpenAI format

### Modified Capabilities

- `go-server`: The HTTP server now registers three new routes (`GET /v1/models`, `POST /v1/chat/completions`) in addition to `/healthz`

## Impact

- No new external Go dependencies required (stdlib `net/http`, `os/exec`, `encoding/json`)
- Requires `claude` CLI to be installed and on `PATH` at runtime
- Startup time increases slightly due to model discovery probes (one short `claude --print` per alias)
- `cmd/server/main.go` and `internal/` packages are extended; no files are removed
