## Context

The server currently uses Go's standard `flag` package in `cmd/server/main.go`. The binary exposes two flags: `--config` and `--version`. There are no subcommands. Cobra is the de-facto standard CLI framework in the Go ecosystem and ships built-in support for generating shell autocompletion scripts.

## Goals / Non-Goals

**Goals:**
- Replace `flag` with `cobra.Command` for the root command
- Preserve existing flag semantics (`--config`, `--version`)
- Add `completion <shell>` subcommand that writes autocompletion scripts to stdout
- Keep `internal/` packages untouched

**Non-Goals:**
- Adding new subcommands beyond `completion`
- Changing config file format or search logic
- Splitting `main.go` into multiple files (unless it becomes unwieldy)

## Decisions

### 1. Root command is the server; `completion` is a subcommand

The root command (`cobra-cli-autocomplete` binary) runs the server directly when invoked without a subcommand. This preserves backward compatibility — existing scripts that call the binary without a subcommand keep working.

Alternative considered: make `serve` an explicit subcommand. Rejected because it's a breaking change for existing users.

### 2. `--version` stays as a flag on the root command

Cobra supports `--version` natively via `cmd.Version` field. Setting this field makes Cobra automatically handle `--version` and `-v` flags, printing the version string. This matches current behavior.

Alternative: `version` subcommand. Rejected to avoid breaking change.

### 3. Use Cobra's built-in `completion` command

`cobra.Command` provides `GenBashCompletion`, `GenZshCompletion`, `GenFishCompletion`, and `GenPowerShellCompletion` methods. We wire these up under a `completion` subcommand with `<shell>` as a positional argument.

Shell choices: `bash`, `zsh`, `fish`, `powershell`.

### 4. Add `github.com/spf13/cobra` as a direct dependency

No wrapper or abstraction. Import directly in `cmd/server/main.go`.

## Risks / Trade-offs

- [Dependency bloat] Cobra pulls in `github.com/spf13/pflag` → Mitigation: acceptable for a CLI tool; pflag is stable and widely used.
- [Flag name change] `pflag` (used by Cobra) uses `--flag` syntax identical to `flag` package → no user-visible change.

## Migration Plan

1. `go get github.com/spf13/cobra@latest`
2. Rewrite `cmd/server/main.go` with Cobra root + `completion` subcommand
3. Run existing tests to confirm no regressions
4. Update `go.mod` / `go.sum`
