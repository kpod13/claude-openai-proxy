## Context

`internal/autorun/macos.go` writes a LaunchAgent plist and activates it with `launchctl bootstrap gui/$UID <plist>`.

The plist is rendered with **`html/template`**, which HTML-escapes the leading XML declaration: the generated file begins with `&lt;?xml version="1.0" …` instead of `<?xml …`. That is invalid XML, so launchd rejects the file and `launchctl bootstrap` returns `5: Input/output error` (issue #13). Confirmed by dumping the rendered plist (`generatePlist`) — its first line is literally `&lt;?xml` — and by observing that a hand-written, valid plist with the same paths/keys bootstraps successfully.

(An earlier investigation mis-attributed the failure to `KeepAlive=false`; that was a confound — the failing test bootstrapped the corrupted install-written file while the passing tests used hand-written valid XML.)

Secondary issues: launchd gives the job a minimal `PATH` (`/usr/bin:/bin:/usr/sbin:/sbin`) lacking the `claude` CLI, and `Install` leaves the plist behind when bootstrap fails.

## Goals / Non-Goals

**Goals:**
- `autorun install` succeeds on macOS and registers a working autostart entry.
- The launched server can find the `claude` CLI under launchd.
- Install is atomic: no plist left behind if activation fails.
- Spec scenarios match the implementation (`bootstrap`/`bootout`, not `load`/`unload`).

**Non-Goals:**
- Changing Linux/Windows behavior.
- Changing how the binary path is resolved (`os.Executable()` stays).
- Bundling/locating `claude` for the user beyond extending `PATH`.

## Decisions

### D0 (primary): Render the plist with `text/template`, not `html/template`
`html/template` is for HTML and corrupts the `<?xml` declaration. Switch to `text/template`, which emits the template verbatim, and explicitly XML-escape the only user-controlled values (binary path, label) with `encoding/xml`'s `EscapeText` via a template `xml` func. This keeps the existing path-escaping behavior (e.g. `&`→`&amp;`) while producing a valid plist.
- *Alternative*: keep `html/template` and prepend the `<?xml` line outside the template. Rejected — fragile; `html/template` could still mis-escape other content, and it's semantically the wrong tool.
- *Alternative*: build the plist with `howett.net/plist` or `encoding/xml` structs. Rejected — adds a dependency / more churn for a small static document.

### D1: Add `EnvironmentVariables → PATH` to the plist
Set `PATH` to `/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin` so the agent's process can locate `claude` (Homebrew arm64, Homebrew Intel/`/usr/local`, system). This addresses the actual reason the server exits non-zero under launchd.
- *Alternative*: resolve the absolute `claude` path at install time and inject it. Rejected — the proxy discovers `claude` via `PATH`/exec internally; a broad `PATH` is simpler and survives `claude` upgrades. (A future enhancement could prepend the directory of a `claude` found at install time.)

### D2: Run as a keep-alive agent (`KeepAlive=true`)
The proxy is a long-running HTTP server, so the agent should be restarted if it dies. Switch `KeepAlive` from `false` to `true`. launchd's built-in respawn throttle (~10s) prevents a tight crash-loop if `claude` is genuinely unavailable.
- *Note*: this is a correctness/robustness improvement, not the EIO fix (that's D0).

### D3: Make `Install` atomic on bootstrap failure
If `launchctl bootstrap` returns an error, remove the plist that was just written before returning the error, so a retry starts clean and no orphan file is left in `~/Library/LaunchAgents`.
- *Note*: the config-write step already runs after bootstrap; with bootstrap now succeeding, the config is written as specified. Config writing remains idempotent (only when absent).

### D4: Correct the spec scenarios
The `autorun` spec says install runs `launchctl load` and uninstall runs `launchctl unload`. The implementation uses the modern `launchctl bootstrap`/`bootout`. Update the scenarios to match; `bootstrap`/`bootout` are the correct, non-deprecated calls.

## Risks / Trade-offs

- **`claude` still not on the extended PATH** → If the user installed `claude` somewhere exotic, the agent still can't start; with `KeepAlive=true` launchd will throttle-respawn it. Mitigation: documented `PATH` covers the common Homebrew/system locations; install no longer fails outright.
- **Hard-coded PATH list** → Could omit a user's custom location. Mitigation: keep the list to well-known dirs; revisit if reports arise.
- **Behavior change for anyone who relied on once-only run** → none expected; the entry is meant to keep the server running.
