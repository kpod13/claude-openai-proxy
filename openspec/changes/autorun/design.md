## Context

The proxy is a single Go binary. It needs to register itself as a user-level autostart entry on four OS families. All autostart mechanisms must run the process under the current user's session (not as root/system), and must survive user logout/login cycles.

## Goals / Non-Goals

**Goals:**
- `autorun install`: register the binary as a user-level autostart entry, write user config to default location.
- `autorun uninstall`: remove the autostart entry.
- Support macOS, Linux, Windows, FreeBSD.
- No root/admin privileges required.

**Non-Goals:**
- System-level (global) service installation.
- Service management beyond install/uninstall (start/stop/status out of scope).
- GUI tray icon or desktop notifications.
- Non-login triggers (e.g., boot without session).

## Decisions

### Platform backends via interface

**Decision**: Define a `Backend` interface in `internal/autorun/`:

```go
type Backend interface {
    Install(cfg InstallConfig) error
    Uninstall() error
}
```

`InstallConfig` carries the binary path, args, and display label. Each platform implements the interface. `New()` returns the correct backend for `runtime.GOOS`.

**Rationale**: Keeps platform logic isolated, independently testable, and easy to extend. Build tags are NOT used — all backends compile on all platforms, selected at runtime. This simplifies cross-compilation and tests.

### Platform mechanisms

| OS | Mechanism | Location |
|---|---|---|
| macOS | launchd user agent | `~/Library/LaunchAgents/com.claude-openai-proxy.plist` |
| Linux | systemd user unit | `~/.config/systemd/user/claude-openai-proxy.service` |
| Windows | Registry `HKCU\...\Run` | Key: `claude-openai-proxy` |
| FreeBSD | cron `@reboot` | User crontab (edited via `crontab -l` / `crontab -`) |

**macOS**: Write plist, then run `launchctl load` to activate immediately. Uninstall: `launchctl unload` then delete plist.

**Linux**: Write `.service` file, then `systemctl --user enable --now`. Uninstall: `systemctl --user disable --now` then delete file. If systemd is not available, fall back to `~/.config/autostart/<name>.desktop` (XDG autostart).

**Windows**: Use `golang.org/x/sys/windows/registry` (already available via indirect deps or added directly) to write/delete the `Run` key.

**FreeBSD**: Parse current crontab, add/remove `@reboot /path/to/binary` line, write back with `crontab -`.

### Binary path resolution

`os.Executable()` resolves the current binary path at install time. This is saved into the autostart entry so the exact binary that ran `install` is what starts on login.

### User config

`install` writes the user config to `~/.claude-code-openai-server.yaml` only if it does not already exist (avoids overwriting an existing config). If it does exist, it is left untouched and a notice is printed. The auto-started binary finds the config via the existing default search path — no `--config` flag needed.

### Args stored in autostart entry

The autostart entry runs the binary with no extra args (it will find the config automatically). Optional: `--quiet` to suppress login-time log noise — configurable via a flag on `autorun install`.

## Risks / Trade-offs

- **Linux without systemd**: XDG autostart fallback (`.desktop` file) requires a desktop environment. Headless Linux servers have no clean user-login hook without systemd. → Documented limitation.
- **Windows registry writes**: Requires `golang.org/x/sys` dependency. → Small, well-maintained, widely used.
- **Binary path moves after install**: Autostart entry points to the old path. → `install` prints a reminder; user must re-run `install` after upgrading/moving the binary.
- **FreeBSD crontab parse fragility**: Crontab may have unusual formatting. → Parse only `@reboot` lines added by this tool (tagged with a comment marker).
