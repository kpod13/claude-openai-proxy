# claude-openai-proxy

An OpenAI-compatible HTTP proxy backed by the [Claude CLI](https://claude.ai/code).

Translates `/v1/chat/completions` and `/v1/models` requests into Claude subprocess calls, so any OpenAI-compatible client (LangChain, openai-python, Cursor, etc.) can use Claude without changing a line of code.

## Features

- Drop-in OpenAI API compatibility (`/v1/chat/completions`, `/v1/models`)
- Streaming and non-streaming responses
- OpenAI-compatible rate limiting with `x-ratelimit-*` headers
- Shell autocompletion (bash, zsh, fish, PowerShell)
- Structured logging (plain text or JSON)

## Requirements

- [Claude CLI](https://claude.ai/code) installed and authenticated (`claude --version`)

> **Building from source** also requires Go 1.26+.

## Installation

**From source:**

```bash
go install github.com/timur/claude-code-openai-server/cmd/server@latest
```

**Pre-built binaries** are available on the [Releases](../../releases) page for Linux, macOS, and Windows (amd64/arm64).

## Quick Start

```bash
# Start the proxy on the default address (127.0.0.1:8080)
claude-openai-proxy

# Point your OpenAI client at it
export OPENAI_BASE_URL=http://127.0.0.1:8080/v1
export OPENAI_API_KEY=any   # value is ignored
```

## Configuration

The server looks for a config file in this order:

1. Path from `--config` flag
2. `~/.claude-code-openai-server.yaml`
3. `/etc/claude-code-openai-server/config.yaml`
4. Built-in defaults

**Example config:**

```yaml
# TCP address to listen on (default: 127.0.0.1:8080)
listen: "0.0.0.0:8080"

# Claude model aliases to discover at startup
aliases:
  - opus
  - sonnet
  - haiku

# Rate limiting (disabled by default; 0 = unlimited)
rate_limit:
  requests_per_minute: 60
  tokens_per_minute: 100000
```

**CLI flags:**

```
--config string       path to config file
--verbose             enable debug-level log output
--quiet               suppress all log output
--log-format string   log format: plain or json (default: plain)
```

## Rate Limiting

Rate limiting is **disabled by default**. When enabled, the proxy enforces per-API-key limits using fixed 1-minute windows and returns OpenAI-compatible headers on every `/v1/chat/completions` response:

| Header | Description |
|---|---|
| `x-ratelimit-limit-requests` | Configured RPM limit |
| `x-ratelimit-limit-tokens` | Configured TPM limit |
| `x-ratelimit-remaining-requests` | Requests left in current window |
| `x-ratelimit-remaining-tokens` | Tokens left in current window |
| `x-ratelimit-reset-requests` | Time until requests window resets |
| `x-ratelimit-reset-tokens` | Time until tokens window resets |

When a limit is exceeded the proxy returns `429 Too Many Requests` with a `Retry-After` header and an OpenAI-compatible JSON error body:

```json
{
  "error": {
    "message": "Rate limit reached: 60 requests per minute. Please retry after 30s.",
    "type": "requests",
    "param": null,
    "code": "rate_limit_exceeded"
  }
}
```

The rate limiter is keyed on the `Authorization: Bearer <token>` header value. Unauthenticated requests share a single anonymous bucket.

## Development

```bash
make build   # build binary to bin/server
make run     # build and run
make test    # run tests
make lint    # run golangci-lint
```

## License

[MIT](LICENSE)
