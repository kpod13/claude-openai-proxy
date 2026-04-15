## Context

golangci-lint v2.11.4 is installed. The project uses Go 1.22+, stdlib only, no frameworks. The codebase is small (~400 LOC across `internal/proxy` and `cmd/server`). The v2 config format uses top-level `linters:` with inline settings (not the legacy `linters-settings:` key).

## Goals / Non-Goals

**Goals:**
- Enable a curated set of linters covering correctness, modern idioms, error-handling, and style
- Zero warnings on the existing codebase (fix violations, don't suppress)
- `make lint` runs the full check in CI-friendly mode

**Non-Goals:**
- Enabling every available linter (many are project-type-specific: promlinter, spancheck, ginkgolinter, etc.)
- Auto-fix pass (lint is check-only; fixes are applied manually or via `--fix` when needed)
- Per-file nolint directives (fix the code instead)

## Decisions

### Linter selection strategy
Group linters into tiers:

**Correctness** — bugs and unsafe patterns:
`errcheck`, `govet`, `staticcheck`, `unused`, `bodyclose`, `nilerr`, `nilnil`, `nilnesserr`, `errorlint`, `durationcheck`, `makezero`, `reassign`, `forcetypeassert`, `noctx`, `fatcontext`

**Modern Go idioms** — features available since Go 1.21/1.22:
`modernize`, `intrange`, `copyloopvar`, `exptostd`, `usestdlibvars`, `perfsprint`, `mirror`, `prealloc`

**Error handling discipline**:
`err113`, `wrapcheck`, `errname`

**Style and maintainability**:
`revive`, `gocritic`, `misspell`, `whitespace`, `nakedret`, `unconvert`, `unparam`, `wastedassign`, `godot`, `nolintlint`, `recvcheck`, `sloglint`, `loggercheck`

**Security**:
`gosec`

### Excluded linters (with rationale)
- `exhaustruct` — requires all struct fields initialised; too noisy for stdlib structs (e.g., `http.Server`)
- `gochecknoglobals` / `gochecknoinits` — idiomatic Go uses both legitimately
- `varnamelen` — single-letter variables are idiomatic in Go (loop vars, receivers)
- `wsl_v5` / `nlreturn` — enforce blank-line style that conflicts with gofmt norms
- `testpackage` — we use `package proxy` (white-box) tests deliberately
- `paralleltest` / `tparallel` — not applicable at current test scale
- `funlen` / `cyclop` / `gocognit` — complexity thresholds produce too many false positives at this scale
- `lll` — line length is enforced by editor, not CI
- `mnd` — magic numbers are fine for HTTP status codes and small constants
- `noinlineerr` — inline `if err := ...; err != nil` is idiomatic Go

### `wrapcheck` scope
Only applied to errors returned from `os/exec` and `net/http` package boundaries; internal package errors are exempt.

### `revive` rules
Enable `exported`, `var-naming`, `error-return`, `error-naming`, `unused-parameter`, `empty-block`, `superfluous-else`, `time-equal`.

### `gocritic` checkers
Enable the `diagnostic`, `style`, and `performance` tag groups.

### Makefile target
```makefile
lint:
    golangci-lint run ./...
```

## Risks / Trade-offs

- **New violations in existing code** → fix them as part of this change (task 3)
- **`err113` requires wrapping sentinel errors** → the codebase uses `fmt.Errorf` already; compliance is low-effort
- **`wrapcheck` on exec errors** → already wrapped with `fmt.Errorf("claude: %w", err)`
