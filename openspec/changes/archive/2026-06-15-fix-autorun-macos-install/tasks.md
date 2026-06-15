## 1. Fix the macOS LaunchAgent plist

- [x] 1.0 PRIMARY: render the plist with `text/template` instead of `html/template` (which escaped `<?xml`→`&lt;?xml`, producing invalid XML); XML-escape the binary path/label via `encoding/xml`
- [x] 1.1 In `internal/autorun/macos.go`, add `EnvironmentVariables` with `PATH=/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin` to the plist template
- [x] 1.2 Change `KeepAlive` from `false` to `true` (the proxy is a long-running server)
- [x] 1.3 Keep `RunAtLoad` and the `os.Executable()` binary path as-is

## 2. Make install atomic

- [x] 2.1 In `macosBackend.Install`, on `launchctl bootstrap` error, remove the plist that was just written before returning the error
- [x] 2.2 Verify the config-write step still runs only after a successful bootstrap and remains idempotent (write only when absent)

## 3. Tests

- [x] 3.1 Update/extend `internal/autorun/macos_test.go` to assert the generated plist contains the `EnvironmentVariables` `PATH` and `KeepAlive` enabled
- [x] 3.2 Add a test that `Install` removes the plist when the (mocked) `launchctl bootstrap` command fails
- [x] 3.3 Run `go test ./internal/autorun/...` and `go vet ./...`

## 4. Spec sync

- [x] 4.1 Confirm `uninstall` already uses `launchctl bootout` in code (spec scenario corrected from `unload` to `bootout`)
- [x] 4.2 Run `openspec validate fix-autorun-macos-install`

## 5. Manual verification (macOS)

- [x] 5.1 `claude-openai-proxy autorun install` exits 0 and the agent appears in `launchctl list`
- [x] 5.2 `~/.claude-code-openai-server.yaml` is written on first install; existing config is left untouched
- [x] 5.3 `claude-openai-proxy autorun uninstall` removes the agent and plist; re-running is a no-op (idempotent)
