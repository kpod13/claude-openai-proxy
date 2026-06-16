# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Repo Is

`claude-openai-proxy` is a Go HTTP server that exposes an **OpenAI-compatible API** (`/v1/chat/completions`, `/v1/models`) and serves it by shelling out to the **Claude CLI**. Any OpenAI-compatible client (LangChain, openai-python, Cursor, etc.) can point at it and talk to Claude without code changes.

**Module**: `github.com/kpod13/claude-openai-proxy` (Go 1.26+).

**Runtime dependency**: the `claude` CLI must be installed, authenticated, and on `PATH`. The proxy invokes it as a subprocess — there is no Anthropic API key handling here.

## Build / Test / Lint

Use the Makefile (no other build system):

```bash
make build   # go build -o bin/claude-openai-proxy ./cmd/claude-openai-proxy
make run     # build and run on the default address
make test    # go test ./...
make lint    # golangci-lint run ./...
make ci      # run GitHub Actions locally via act
```

Lint config is `.golangci.yml` (strict; e.g. `wsl_v5` whitespace rules — keep a blank line before goroutines). CI runs lint + tests on push.

## Architecture

The entrypoint is `cmd/claude-openai-proxy/main.go`, a Cobra CLI (`runServer`, `autorun`, `completion` subcommands). HTTP handling and the Claude bridge live in `internal/`:

| Package | Responsibility |
|---|---|
| `internal/proxy` | HTTP handlers + Claude subprocess bridge. `handler.go` (routing, request/response shaping), `claude.go` (`RunBlocking` / `RunBlockingImages` / `RunStreaming` / `RunStreamingImages`), `models.go`, `types.go`, `image.go`/`debug.go` |
| `internal/config` | YAML config loading, search order, defaults, validation |
| `internal/ratelimit` | Per-API-key fixed-window limiter + middleware, OpenAI `x-ratelimit-*` headers |
| `internal/autorun` | OS-specific user-level autostart (launchd / systemd / registry) |
| `internal/logger` | Structured logging (plain or JSON) |

### How Claude is invoked

`internal/proxy/claude.go` runs `claude` headlessly:

- Non-streaming text: `claude --print --output-format json --model <id> --no-session-persistence`
- Streaming text: `--print --output-format stream-json --verbose ...`
- Image input: adds `--input-format stream-json` (stream-json input requires stream-json output)

Model IDs are sanitized (`sanitizeModelID`, letters/digits/hyphens only) before being passed as flags. The `newCommand` factory (`exec.CommandContext`) is swapped out in tests.

**Headless permission caveat**: in `--print` mode there is no TTY, so any tool call that needs an interactive permission prompt blocks forever and the HTTP request hangs. The OpenAI protocol has no channel to relay a permission request, so the policy must be set server-side via `claude` flags (`--permission-mode`, `--allowedTools`, `--add-dir`). See the `add-permission-policy` change for the planned config-driven policy.

## Configuration

Config is YAML, searched in order (first found wins): `--config <path>` → `/etc/claude-code-openai-server/config.yaml` → `~/.claude-code-openai-server.yaml` → built-in defaults. Keys: `listen`, `aliases`, `rate_limit`. See `README.md` and `internal/config/config.go`.

## Specs (OpenSpec)

This repo also tracks behavior with OpenSpec. The `openspec` CLI must be on `PATH` for that workflow.

- Capability specs (source of truth for behavior): `openspec/specs/<capability>/spec.md`
- In-flight changes: `openspec/changes/<name>/` with `proposal.md` → `design.md` → `specs/` → `tasks.md`; archived to `openspec/changes/archive/YYYY-MM-DD-<name>/`
- Slash commands `/opsx:propose|explore|apply|archive` (skills under `.claude/skills/`) drive the lifecycle.

When implementing a change, keep `tasks.md` checkboxes (`- [ ]`/`- [x]`) updated as you go, and sync the affected `openspec/specs/` on archive.

## Conventions

- Task descriptions in `tasks.md` are written in English.
- Do not run `git commit` unless explicitly asked.
- Match the surrounding code style; the linter is strict, so run `make lint` before finishing.
