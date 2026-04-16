## Context

The server uses `log.Println` / `log.Printf` / `log.Fatal` from the standard library. There is no level filtering, no structured output, and no way to silence or increase verbosity at runtime. Go 1.21+ ships `log/slog` in the standard library, providing structured logging with level control and pluggable handlers — no external dependency needed.

## Goals / Non-Goals

**Goals:**
- `internal/logger` package: thin wrapper around `slog.Logger` with `Info`, `Debug`, `Error` methods
- Level control: INFO (default), DEBUG (--verbose), silent (--quiet)
- Format control: text/plain (default) or JSON (--log-format=json)
- Three new root-command flags: `--verbose`, `--quiet`, `--log-format`
- Replace all `log.*` calls in `cmd/server/main.go` with the new logger

**Non-Goals:**
- Log file output / log rotation
- Per-subsystem log levels
- Dynamic level changes at runtime
- Structured fields beyond what slog provides out of the box

## Decisions

### 1. Use `log/slog` from stdlib — no external dependency

`log/slog` (Go 1.21+) provides `TextHandler` (plain) and `JSONHandler` (JSON) out of the box. No third-party package (zap, zerolog, logrus) needed. The project already requires Go 1.26+ so slog is available.

Alternative: zerolog. Rejected — unnecessary dependency for a small CLI tool.

### 2. `internal/logger` wraps a `*slog.Logger` value

The package exposes a `New(level, format, quiet)` constructor and top-level `Info`, `Debug`, `Error` functions that delegate to a package-level logger. This avoids threading a logger through every call site while keeping it testable.

### 3. Levels map: quiet → no output, default → INFO, --verbose → DEBUG

`--quiet` sets the handler to `io.Discard`. `--verbose` sets level to `slog.LevelDebug`. Default is `slog.LevelInfo`.

`--quiet` takes precedence over `--verbose` if both are set.

### 4. `--log-format` flag: `plain` (default) | `json`

`plain` → `slog.NewTextHandler(os.Stderr, ...)` — human-readable key=value output.  
`json` → `slog.NewJSONHandler(os.Stderr, ...)` — machine-readable JSON lines.

Output goes to stderr (same as the current `log` package default).

### 5. Replace `log.Fatal` with error returns through Cobra

`log.Fatal` calls `os.Exit(1)` directly, bypassing deferred cleanup and Cobra's error handling. Replacing it with `return err` inside `RunE` lets Cobra print the error and exit cleanly.

## Risks / Trade-offs

- [slog text format differs from log package format] → Timestamps and format change slightly. Acceptable — no users depend on log format parsing.
- [--quiet silences error logs too] → By design; matches conventional `--quiet` semantics. Errors are still returned as exit codes.

## Migration Plan

1. Create `internal/logger/logger.go`
2. Add flags to root Cobra command; build logger after flag parse in `PersistentPreRunE`
3. Replace `log.*` calls in `main.go`
4. Run tests + linter
