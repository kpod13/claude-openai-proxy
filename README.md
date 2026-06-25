# claude-openai-proxy

An OpenAI-compatible HTTP proxy backed by the [Claude CLI](https://claude.ai/code).

Translates `/v1/chat/completions` and `/v1/models` requests into Claude subprocess calls, so any OpenAI-compatible client (LangChain, openai-python, Cursor, etc.) can use Claude without changing a line of code.

## Features

- Drop-in OpenAI API compatibility (`/v1/chat/completions`, `/v1/models`)
- Streaming and non-streaming responses
- Multimodal image input (OpenAI `image_url` parts — base64 `data:` URIs and `http(s)` URLs)
- OpenAI-compatible rate limiting with `x-ratelimit-*` headers
- User-level autostart (`autorun install`) for macOS, Linux, Windows
- Shell autocompletion (bash, zsh, fish, PowerShell)
- Structured logging (plain text or JSON)

## Requirements

- [Claude CLI](https://claude.ai/code) installed and authenticated (`claude --version`)

> **Building from source** also requires Go 1.26+.

## Installation

**Homebrew (macOS / Linux):**

```bash
brew tap kpod13/tap
brew install claude-openai-proxy
```

Upgrade later with `brew upgrade claude-openai-proxy`. If you've enabled autostart (`autorun install`), the upgrade restarts the running agent automatically so it picks up the new binary.

**From source:**

```bash
go install github.com/kpod13/claude-openai-proxy/cmd/claude-openai-proxy@latest
```

**Pre-built archives** are available on the [Releases](../../releases) page for Linux, macOS, Windows, and FreeBSD (amd64/arm64). Each release ships per-platform `.tar.gz`/`.zip` archives plus a `checksums.txt`; download the archive for your platform and extract the `claude-openai-proxy` binary onto your `PATH`.

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

## Autorun

To start the proxy automatically when you log in:

```bash
# Register as user-level autostart entry and write default config
claude-openai-proxy autorun install

# Remove the autostart entry
claude-openai-proxy autorun uninstall
```

The mechanism is OS-specific and requires no root privileges:

| OS      | Mechanism                                                                                                     |
|---------|---------------------------------------------------------------------------------------------------------------|
| macOS   | `~/Library/LaunchAgents/com.claude-openai-proxy.plist` (launchd)                                              |
| Linux   | `~/.config/systemd/user/claude-openai-proxy.service` (systemd), falls back to `~/.config/autostart/*.desktop` |
| Windows | `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` (registry)                                               |

On first install, a default config is written to `~/.claude-code-openai-server.yaml` if it doesn't already exist. If you move or upgrade the binary, re-run `autorun install` to update the entry.

## Rate Limiting

Rate limiting is **disabled by default**. When enabled, the proxy enforces per-API-key limits using fixed 1-minute windows and returns OpenAI-compatible headers on every `/v1/chat/completions` response:

| Header                           | Description                       |
|----------------------------------|-----------------------------------|
| `x-ratelimit-limit-requests`     | Configured RPM limit              |
| `x-ratelimit-limit-tokens`       | Configured TPM limit              |
| `x-ratelimit-remaining-requests` | Requests left in current window   |
| `x-ratelimit-remaining-tokens`   | Tokens left in current window     |
| `x-ratelimit-reset-requests`     | Time until requests window resets |
| `x-ratelimit-reset-tokens`       | Time until tokens window resets   |

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

## Permissions

The proxy runs `claude` headlessly (`--print`), where there is no terminal to answer interactive permission prompts. If a request causes Claude to use a tool that needs approval (e.g. `Write`, `Bash`), the subprocess blocks waiting for an answer that can never come and the request hangs. The OpenAI protocol has no way to relay a permission request, so the policy must be set server-side via the optional `permission` config block:

```yaml
permission:
  mode: default                 # acceptEdits | auto | bypassPermissions | default | dontAsk | plan
  allowed_tools:                # tool specs allowed without prompting
    - Write
    - Edit
    - "Bash(git *)"
  disallowed_tools: []          # tool specs to deny
  add_dirs:                     # extra directories tools may access
    - /srv/work
```

`mode` accepts the values from the `claude` CLI `--permission-mode` flag:

| `mode`              | Behavior                                                                                                                      |
|---------------------|-------------------------------------------------------------------------------------------------------------------------------|
| `default`           | Ask for permission as usual. Headless tool calls that need approval **hang** — this is the safe, behavior-preserving default. |
| `acceptEdits`       | Auto-accept file edits, still ask for other tools.                                                                            |
| `plan`              | Planning mode; no tools are executed.                                                                                         |
| `dontAsk`           | Do not prompt; relies on `allowed_tools` / `disallowed_tools`.                                                                |
| `auto`              | Let Claude decide automatically.                                                                                              |
| `bypassPermissions` | Skip all permission checks (**dangerous** — see warning below).                                                               |

**Under `mode: default` only** (the default, including when no `permission` block is set), the proxy substitutes a hang-free default for any tool list left empty, **independently per section**:

- empty `allowed_tools` → `WebSearch`, `WebFetch` (low-risk web tools, pre-approved so they don't prompt)
- empty `disallowed_tools` → the other permission-requiring tools: `Artifact`, `Bash`, `Edit`, `ExitPlanMode`, `Monitor`, `NotebookEdit`, `PowerShell`, `ShareOnboardingGuide`, `Skill`, `Workflow`, `Write`

Read-only tools (`Read`, `Grep`, `Glob`, …) are left available — they never prompt, so they never hang. A non-empty list is used verbatim and is **not** merged with its default, so set `allowed_tools` / `disallowed_tools` explicitly to override these defaults (an empty inline list like `disallowed_tools: []` counts as empty and still gets the default — list at least one entry to opt out).

Any **other** `mode` (`acceptEdits`, `auto`, `dontAsk`, `bypassPermissions`, `plan`) is treated as an explicit choice: empty tool lists are left empty, so the deny-list defaults can't silently override the mode you picked. Combine such a mode with your own `allowed_tools` / `disallowed_tools` as needed.

The whole block is validated at startup — an unknown `mode`, a malformed tool spec, a blank entry, or any entry beginning with `-` fails fast before the server binds. Tool specs follow the `claude` format: `ToolName` or `ToolName(rule)` (e.g. `Write`, `Bash(git *)`, `mcp__server__tool`).

> ⚠️ **Security:** every OpenAI request becomes tool execution on the host. `mode: bypassPermissions` (or the equivalent CLI flag) is effectively unauthenticated remote code execution for anyone who can reach the listener — especially dangerous with a `0.0.0.0` bind. Prefer a narrow `allowed_tools` list plus `add_dirs` scoped to an isolated working directory.

## Development

```bash
make build   # build binary to bin/claude-openai-proxy
make run     # build and run
make test    # run tests
make lint    # run golangci-lint
```

## Releasing (maintainers)

Releases run from `.github/workflows/release.yml`. Pushing a `v*.*.*` tag runs lint and tests, builds the cross-platform binaries, packages them into `.tar.gz`/`.zip` archives, publishes a GitHub Release with `checksums.txt`, then regenerates the Homebrew formula (`scripts/update-formula.sh`) and pushes it to the [`kpod13/homebrew-tap`](https://github.com/kpod13/homebrew-tap) repository.

Pushing the formula to that separate repo requires the `HOMEBREW_TAP_TOKEN` Actions secret (the default `GITHUB_TOKEN` cannot push to another repository):

- **Purpose:** lets the release workflow commit `Formula/claude-openai-proxy.rb` to `kpod13/homebrew-tap`.
- **Scope:** a fine-grained personal access token limited to the `kpod13/homebrew-tap` repository with **Contents: Read and write**.
- **Rotation:** regenerate the token before it expires (or if leaked) and update the `HOMEBREW_TAP_TOKEN` secret under **Settings → Secrets and variables → Actions**. A failed/expired token surfaces as a failed `release` job; the GitHub Release still succeeds but the formula will not update.

## License

[MIT](LICENSE)
