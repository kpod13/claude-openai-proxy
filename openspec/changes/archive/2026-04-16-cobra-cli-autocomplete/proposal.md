## Why

The server binary currently uses the standard `flag` package, which provides limited CLI ergonomics — no subcommands, no built-in help formatting, and no shell autocompletion. Migrating to [Cobra](https://github.com/spf13/cobra) unlocks a richer CLI structure and first-class autocompletion generation for bash, zsh, fish, and PowerShell.

## What Changes

- Replace `flag`-based CLI in `cmd/server/main.go` with a Cobra root command
- Keep existing flags (`--config`, `--version`) as Cobra persistent/root flags
- Add `completion` subcommand that generates shell autocompletion scripts (bash, zsh, fish, powershell)
- Add `cobra` dependency to `go.mod` / `go.sum`

## Capabilities

### New Capabilities
- `cobra-cli`: Root Cobra command wiring — flags, version output, server start logic
- `shell-completion`: `completion <shell>` subcommand that prints autocompletion scripts to stdout

### Modified Capabilities
<!-- No existing spec-level behavior changes -->

## Impact

- `cmd/server/main.go` — full rewrite around Cobra
- `go.mod` / `go.sum` — add `github.com/spf13/cobra`
- No changes to `internal/` packages
- Binary interface change: `--version` flag becomes `version` subcommand or stays as flag (TBD in design); `completion` is a new subcommand
