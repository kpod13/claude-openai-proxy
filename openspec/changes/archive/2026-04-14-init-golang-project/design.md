## Context

The repository is a Claude Code extension (skills + OpenSpec workflow config) with no runnable application code. Adding a Go project lays the groundwork for a deployable server that can be extended with actual business logic. The project structure must coexist cleanly with existing `.claude/` and `openspec/` directories.

## Goals / Non-Goals

**Goals:**
- A valid, buildable Go module at the repo root
- A minimal HTTP server that responds to requests (proves the scaffold works)
- Standard Go layout (`cmd/server/`, `internal/`) for future growth
- A `Makefile` covering `build`, `run`, and `test` targets
- A Go-appropriate `.gitignore`

**Non-Goals:**
- Any business logic, routing framework, or middleware
- Database setup, configuration management, or Docker/container support
- CI/CD pipeline changes
- Altering existing `.claude/` or `openspec/` files

## Decisions

### Module path
Use `github.com/timur/claude-code-openai-server` as the module path, matching the repository name. This is conventional and avoids a placeholder that would need renaming later.

**Alternative considered**: `example.com/app` — rejected as it signals throwaway scaffolding.

### Entry point location
Place `main.go` under `cmd/server/` rather than at the repo root. Go convention places runnable binaries under `cmd/<name>/`, keeping the root clean and supporting multiple binaries later.

**Alternative considered**: root-level `main.go` — simpler initially but conflicts with idiomatic Go layout.

### HTTP server
Use `net/http` from the standard library only. A single `/healthz` handler is sufficient to prove the server boots and listens.

**Alternative considered**: a third-party router (chi, gin) — premature at scaffolding stage; frameworks can be added when actual routes are needed.

### Makefile vs shell scripts
A `Makefile` is universally available on macOS/Linux and requires no additional tooling. Three targets: `build`, `run`, `test`.

## Risks / Trade-offs

- **Go version lock** → Pin `go 1.22` in `go.mod`; upgrading later is a one-line change.
- **Module path may not match actual remote** → Acceptable at init stage; can be renamed when the repo is pushed to a specific host.

## Migration Plan

No existing code is changed. New files are additive. No deployment steps required for scaffolding.
