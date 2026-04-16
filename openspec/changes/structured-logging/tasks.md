## 1. Logger Package

- [ ] 1.1 Create `internal/logger/logger.go` with `New(verbose, quiet bool, format string) *slog.Logger` constructor
- [ ] 1.2 Wire `TextHandler` for `plain` format and `JSONHandler` for `json` format, both writing to stderr
- [ ] 1.3 Set level to `slog.LevelDebug` when verbose=true, `slog.LevelInfo` otherwise; redirect to `io.Discard` when quiet=true
- [ ] 1.4 Write `internal/logger/logger_test.go` covering: info-suppresses-debug, verbose-emits-debug, quiet-suppresses-all, quiet-overrides-verbose

## 2. CLI Flags

- [ ] 2.1 Add `--verbose` (bool), `--quiet` (bool), and `--log-format` (string, default `plain`) persistent flags to the root Cobra command
- [ ] 2.2 Build the logger in `PersistentPreRunE` after flags are parsed and store it in a variable accessible to `RunE`

## 3. Replace log calls

- [ ] 3.1 Replace all `log.Println` / `log.Printf` calls in `cmd/server/main.go` with `logger.Info` / `logger.Debug`
- [ ] 3.2 Remove any remaining `log.Fatal` calls — ensure errors are returned through Cobra's `RunE` instead

## 4. Verification

- [ ] 4.1 Run `go build ./...` — no compile errors
- [ ] 4.2 Run `go test ./...` — all tests pass
- [ ] 4.3 Run `golangci-lint run ./...` — no lint issues
