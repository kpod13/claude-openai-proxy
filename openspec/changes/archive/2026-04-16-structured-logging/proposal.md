## Why

The server currently uses the standard `log` package with no level filtering or structured output, making it hard to control verbosity in production and difficult to parse logs programmatically. Adding structured logging with level control and format selection improves operability.

## What Changes

- Introduce a `logger` internal package wrapping `log/slog` with level and format control
- Add `--verbose` CLI flag: enables `DEBUG`-level output (verbose)
- Add `--quiet` CLI flag: disables all log output
- Add `--log-format` CLI flag: selects output format (`plain` or `json`); default `plain`
- Default log level: `INFO`
- Replace all `log.Println` / `log.Printf` / `log.Fatal` calls in `cmd/server/main.go` with the new logger
- **BREAKING**: `log.Fatal` replaced by returning errors through Cobra — process exit code unchanged

## Capabilities

### New Capabilities
- `logging`: Logger package with level filtering (debug/info/error), format selection (plain/json), quiet mode, and CLI flag wiring

### Modified Capabilities
- `cobra-cli`: Root command gains `--verbose`, `--quiet`, and `--log-format` flags

## Impact

- New package: `internal/logger/`
- `cmd/server/main.go` — replace `log` calls, add flags
- `openspec/specs/cobra-cli/spec.md` — delta for new flags
- No changes to `internal/proxy/` or `internal/config/`
