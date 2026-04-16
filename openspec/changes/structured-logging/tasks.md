## 1. Logger Package

- [x] 1.1 Create `internal/logger/logger.go` with `New(verbose, quiet bool, format string) *slog.Logger` constructor
- [x] 1.2 Wire `TextHandler` for `plain` format and `JSONHandler` for `json` format, both writing to stderr
- [x] 1.3 Set level to `slog.LevelDebug` when verbose=true, `slog.LevelInfo` otherwise; redirect to `io.Discard` when quiet=true
- [x] 1.4 Write `internal/logger/logger_test.go` covering: info-suppresses-debug, verbose-emits-debug, quiet-suppresses-all, quiet-overrides-verbose

## 2. CLI Flags

- [x] 2.1 Add `--verbose` (bool), `--quiet` (bool), and `--log-format` (string, default `plain`) persistent flags to the root Cobra command
- [x] 2.2 Build the logger in `PersistentPreRunE` after flags are parsed and store it in a variable accessible to `RunE`

## 3. Replace log calls

- [x] 3.1 Replace all `log.Println` / `log.Printf` calls in `cmd/server/main.go` with `logger.Info` / `logger.Debug`
- [x] 3.2 Remove any remaining `log.Fatal` calls — ensure errors are returned through Cobra's `RunE` instead

## 4. Verification

- [x] 4.1 Run `go build ./...` — no compile errors
- [x] 4.2 Run `go test ./...` — all tests pass
- [x] 4.3 Run `golangci-lint run ./...` — no lint issues
