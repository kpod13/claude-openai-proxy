## Why

There is no way to make the proxy start automatically after user login. Users have to start it manually every time, which breaks integrations that expect the proxy to always be available. Adding autorun support with a single CLI command makes the proxy behave like a proper background service.

## What Changes

- Add `autorun install` CLI subcommand: writes a user-level autostart entry for the current OS and saves the user config file.
- Add `autorun uninstall` CLI subcommand: removes the autostart entry and optionally cleans up the config.
- Add `internal/autorun/` package with platform-specific backends: macOS (launchd), Linux (systemd user), Windows (registry), FreeBSD (cron `@reboot`).
- The autostart unit/entry runs the proxy at **user session level** (not system), triggered after login.
- The user config is saved to `~/.claude-code-openai-server.yaml` (already the default search path) so the auto-started process picks it up without extra flags.

## Capabilities

### New Capabilities

- `autorun`: CLI subcommands and platform-specific backends for user-level autostart provisioning and deprovisioning.

### Modified Capabilities

- `cobra-cli`: Add `autorun` subcommand group (`install` / `uninstall`) to the Cobra command tree.

## Impact

- New package `internal/autorun/` — platform-specific install/uninstall logic, no new external dependencies.
- `cmd/server/main.go` — register `autorun` subcommand.
- Creates/removes OS-specific files or registry entries in the user's home directory.
- No changes to the server's HTTP logic.
