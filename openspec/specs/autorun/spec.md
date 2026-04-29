### Requirement: autorun install provisions user-level autostart
`autorun install` SHALL register the proxy binary as a user-level autostart entry that runs after the current user logs in. The mechanism used SHALL be appropriate for the host OS:
- **macOS**: launchd user agent plist at `~/Library/LaunchAgents/com.claude-openai-proxy.plist`, activated with `launchctl load`.
- **Linux**: systemd user unit at `~/.config/systemd/user/claude-openai-proxy.service`, enabled with `systemctl --user enable --now`. Falls back to XDG autostart (`~/.config/autostart/claude-openai-proxy.desktop`) if systemd is unavailable.
- **Windows**: registry value under `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` with key `claude-openai-proxy`.

The entry SHALL use the absolute path of the currently running binary (`os.Executable()`).
The install command SHALL NOT require root or administrator privileges.

#### Scenario: Install on macOS creates plist and loads agent
- **WHEN** `autorun install` is run on macOS
- **THEN** `~/Library/LaunchAgents/com.claude-openai-proxy.plist` is created and `launchctl load` is executed

#### Scenario: Install on Linux creates systemd user unit
- **WHEN** `autorun install` is run on Linux with systemd available
- **THEN** `~/.config/systemd/user/claude-openai-proxy.service` is created and `systemctl --user enable --now` is executed

#### Scenario: Install on Windows writes registry key
- **WHEN** `autorun install` is run on Windows
- **THEN** the registry key `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\claude-openai-proxy` is set to the binary path

### Requirement: autorun fails on unsupported OS
On operating systems other than macOS, Linux, and Windows, `autorun install` and `autorun uninstall` SHALL exit with a non-zero code and an informative error message.

#### Scenario: Unsupported OS returns error
- **WHEN** `autorun install` or `autorun uninstall` is run on an unsupported OS
- **THEN** the command exits with a non-zero code and an error message identifying the OS as unsupported

### Requirement: autorun uninstall removes the autostart entry
`autorun uninstall` SHALL remove the autostart entry created by `autorun install`, using the same OS-specific mechanism. It SHALL NOT fail if no entry exists (idempotent).

#### Scenario: Uninstall on macOS unloads and removes plist
- **WHEN** `autorun uninstall` is run on macOS and the plist exists
- **THEN** `launchctl unload` is executed and the plist file is deleted

#### Scenario: Uninstall on Linux disables and removes unit
- **WHEN** `autorun uninstall` is run on Linux and the unit exists
- **THEN** `systemctl --user disable --now` is executed and the unit file is deleted

#### Scenario: Uninstall on Windows removes registry key
- **WHEN** `autorun uninstall` is run on Windows
- **THEN** the `claude-openai-proxy` key is removed from `HKCU\...\Run`

#### Scenario: Uninstall is idempotent
- **WHEN** `autorun uninstall` is run and no entry exists
- **THEN** the command exits successfully without error

### Requirement: autorun install writes user config if absent
`autorun install` SHALL write the default config file to `~/.claude-code-openai-server.yaml` if and only if that file does not already exist.

#### Scenario: Config written on first install
- **WHEN** `autorun install` is run and `~/.claude-code-openai-server.yaml` does not exist
- **THEN** a default config file is written to that path

#### Scenario: Existing config is not overwritten
- **WHEN** `autorun install` is run and `~/.claude-code-openai-server.yaml` already exists
- **THEN** the existing file is left unchanged and a notice is printed

### Requirement: autorun uses current binary path
The autostart entry SHALL record the absolute path returned by `os.Executable()` at the time `install` is run. If the binary is moved or upgraded, the user must re-run `install`.

#### Scenario: Binary path recorded correctly
- **WHEN** `autorun install` is run from `/usr/local/bin/claude-openai-proxy`
- **THEN** the autostart entry references `/usr/local/bin/claude-openai-proxy`
