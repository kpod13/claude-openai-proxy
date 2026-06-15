## Why

`autorun install` is broken on macOS ([#13](https://github.com/kpod13/claude-openai-proxy/issues/13)): it fails with `launchctl bootstrap: exit status 5 (Input/output error)` and never registers the autostart entry.

**Root cause:** `internal/autorun/macos.go` renders the plist with `html/template`, which HTML-escapes the leading XML declaration `<?xml … ?>` to `&lt;?xml … ?>`. The resulting plist is **invalid XML**, so launchd refuses to load it and `launchctl bootstrap` returns EIO 5. (Verified: the generated file on disk literally begins with `&lt;?xml`; hand-writing a valid plist with the same paths/keys bootstraps fine.)

Two secondary problems compound it: launchd gives the job a minimal `PATH` that lacks the `claude` CLI (so even a valid plist's server would fail model discovery), and the install is non-atomic (on failure it leaves the plist behind).

## What Changes

- **Primary fix:** render the plist with `text/template` instead of `html/template`, XML-escaping only the user-controlled values (binary path, label) via `encoding/xml`. The `<?xml` declaration is emitted verbatim, producing a valid plist.
- The macOS LaunchAgent SHALL set `EnvironmentVariables → PATH` to include common CLI locations (`/opt/homebrew/bin`, `/usr/local/bin`, `/usr/bin`, `/bin`) so the launched server can find the `claude` CLI.
- The LaunchAgent SHALL run as a long-running agent (`KeepAlive=true`) since the proxy is a persistent server.
- `autorun install` SHALL be atomic on macOS: if `launchctl bootstrap` fails, the plist that was just written SHALL be removed so no partial state is left behind.
- Update the `autorun` spec scenarios that say install runs `launchctl load` / uninstall runs `launchctl unload` to reflect the actual `launchctl bootstrap` / `bootout` mechanism.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `autorun`: macOS install requirements change — the LaunchAgent must provision a working `PATH` and run as a keep-alive agent, install must be atomic on bootstrap failure, and the load/unload scenarios are corrected to `bootstrap`/`bootout`.

## Impact

- **Code**: `internal/autorun/macos.go` (switch `html/template`→`text/template` with `encoding/xml` escaping; add `EnvironmentVariables`/`PATH`; `KeepAlive=true`; `Install` cleans up plist on bootstrap failure). Tests in `internal/autorun/macos_test.go`.
- **Spec**: `openspec/specs/autorun/spec.md` (macOS install requirement + load→bootstrap scenarios).
- **Behavior**: existing macOS users who ran the broken install have no entry; after the fix, re-running `autorun install` succeeds. No change for Linux/Windows.
- **Issue**: closes [#13](https://github.com/kpod13/claude-openai-proxy/issues/13).
