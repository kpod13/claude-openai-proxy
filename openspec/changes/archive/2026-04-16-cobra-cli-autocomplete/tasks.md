## 1. Dependencies

- [x] 1.1 Run `go get github.com/spf13/cobra@latest` to add Cobra to go.mod / go.sum

## 2. Root Command

- [x] 2.1 Rewrite `cmd/server/main.go`: create a Cobra root command with `--config` and `--version` flags
- [x] 2.2 Move server startup logic (config load, model discovery, HTTP server) into the root command's `RunE` function
- [x] 2.3 Set `cmd.Version` so Cobra handles `--version` / `-v` natively
- [x] 2.4 Set `Use`, `Short`, and `Long` fields on the root command so `--help` / `-h` output is descriptive (includes flag list and available subcommands)

## 3. Completion Subcommand

- [x] 3.1 Add `completion` subcommand with `<shell>` positional arg (bash, zsh, fish, powershell)
- [x] 3.2 Wire each shell to its Cobra generation method (`GenBashCompletion`, `GenZshCompletion`, etc.)
- [x] 3.3 Write install instructions into the `completion` subcommand's `Long` help text

## 4. Verification

- [x] 4.1 Run `go build ./...` — ensure no compile errors
- [x] 4.2 Run `go test ./...` — ensure all existing tests pass
- [x] 4.3 Smoke-test `--version`, `--help`, and `completion zsh` manually
