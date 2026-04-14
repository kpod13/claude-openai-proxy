## Why

This repository currently contains only Claude Code extension configuration (skills and commands) with no application code. The goal is to introduce a minimal Go project foundation so the repository can serve as a deployable server, not just a configuration-only extension.

## What Changes

- Add a Go module (`go.mod`) at the repository root
- Add a minimal `main.go` entry point that starts an HTTP server
- Add standard project structure: `cmd/`, `internal/` directories
- Add a `Makefile` with common dev tasks (`build`, `run`, `test`)
- Add a `.gitignore` for Go build artifacts

## Capabilities

### New Capabilities

- `go-server`: A minimal Go HTTP server entry point with module scaffolding and project layout

### Modified Capabilities

<!-- No existing specs are being modified -->

## Impact

- Adds Go as a language dependency (Go 1.22+ toolchain required)
- No existing skills, commands, or OpenSpec configuration is affected
- New files live alongside existing `.claude/` and `openspec/` directories
