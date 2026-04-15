## 1. Dependency

- [x] 1.1 Add `gopkg.in/yaml.v3` to `go.mod` via `go get`

## 2. Config package

- [x] 2.1 Create `internal/config/config.go` with `Config` struct (`Listen string`, `Aliases []string`) and `Load(path string) (*Config, error)` function
- [x] 2.2 Implement `defaultConfig()` returning `Config{Listen: "127.0.0.1:8080", Aliases: []string{"opus", "sonnet", "haiku"}}`
- [x] 2.3 Implement config search: explicit path → `/etc/claude-code-openai-server/config.yaml` → `~/.claude-code-openai-server.yaml` → defaults
- [x] 2.4 Return error when `--config` path is given but file does not exist

## 3. Wire config into main

- [x] 3.1 Add `--config` flag to `cmd/server/main.go` using `flag` package
- [x] 3.2 Call `config.Load` at startup and pass `cfg.Listen` to `http.Server.Addr`
- [x] 3.3 Pass `cfg.Aliases` to `proxy.Discover` instead of the hardcoded slice

## 4. Tests

- [x] 4.1 Add unit tests for `Load`: valid YAML overrides defaults, partial YAML merges, missing explicit path errors, no file found uses defaults

## 5. Verification

- [x] 5.1 Run `go build ./...` — exits 0
- [x] 5.2 Run `go test ./...` — all tests pass
- [x] 5.3 Run `golangci-lint run ./...` — exits 0 with no issues
