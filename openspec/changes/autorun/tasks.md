## 1. Backend Interface & Shared Types

- [ ] 1.1 Create `internal/autorun/autorun.go`: define `Backend` interface (`Install(InstallConfig) error`, `Uninstall() error`) and `InstallConfig` struct
- [ ] 1.2 Create `internal/autorun/new.go`: implement `New() Backend` that selects the correct backend based on `runtime.GOOS`, returns error for unsupported OS

## 2. macOS Backend

- [ ] 2.1 Create `internal/autorun/macos.go`: implement `macosBackend` with plist generation at `~/Library/LaunchAgents/com.claude-openai-proxy.plist`
- [ ] 2.2 `Install`: write plist, run `launchctl load <path>`
- [ ] 2.3 `Uninstall`: run `launchctl unload <path>`, delete plist (idempotent)

## 3. Linux Backend

- [ ] 3.1 Create `internal/autorun/linux.go`: implement `linuxBackend`, try systemd first then XDG fallback
- [ ] 3.2 Systemd path: write `~/.config/systemd/user/claude-openai-proxy.service`, run `systemctl --user enable --now`
- [ ] 3.3 XDG fallback: write `~/.config/autostart/claude-openai-proxy.desktop`
- [ ] 3.4 `Uninstall`: disable+stop unit and delete file (or delete `.desktop`); idempotent

## 4. Windows Backend

- [ ] 4.1 Add `golang.org/x/sys` dependency (`go get golang.org/x/sys`)
- [ ] 4.2 Create `internal/autorun/windows.go`: implement `windowsBackend` using `golang.org/x/sys/windows/registry`
- [ ] 4.3 `Install`: write `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\claude-openai-proxy`
- [ ] 4.4 `Uninstall`: delete the registry value; idempotent if key absent

## 5. FreeBSD Backend

- [ ] 5.1 Create `internal/autorun/freebsd.go`: implement `freebsdBackend` using crontab
- [ ] 5.2 `Install`: read current crontab (`crontab -l`), append `@reboot <path> # claude-openai-proxy`, write back (`crontab -`)
- [ ] 5.3 `Uninstall`: read crontab, remove lines tagged `# claude-openai-proxy`, write back; idempotent

## 6. User Config Helper

- [ ] 6.1 Add `WriteDefaultConfigIfAbsent(path string) (created bool, err error)` in `internal/autorun/config.go`: writes default YAML config to `~/.claude-code-openai-server.yaml` only if the file does not exist

## 7. CLI Wiring

- [ ] 7.1 Create `internal/autorun/cmd.go` (or add to `cmd/server/main.go`): build `autorunCmd` Cobra command with `install` and `uninstall` subcommands
- [ ] 7.2 `install` subcommand: resolve binary path via `os.Executable()`, call `Backend.Install()`, call `WriteDefaultConfigIfAbsent`, print confirmation
- [ ] 7.3 `uninstall` subcommand: call `Backend.Uninstall()`, print confirmation
- [ ] 7.4 Register `autorunCmd` on the root Cobra command in `main.go`

## 8. Tests

- [ ] 8.1 Unit tests for `WriteDefaultConfigIfAbsent`: file absent (creates), file present (skips)
- [ ] 8.2 Unit tests for plist generation (macOS backend): verify plist XML output
- [ ] 8.3 Unit tests for systemd unit file generation (Linux backend): verify `.service` content
- [ ] 8.4 Unit tests for crontab edit logic (FreeBSD backend): add line, remove line, idempotent remove
