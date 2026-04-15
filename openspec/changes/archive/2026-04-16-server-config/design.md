## Context

`cmd/server/main.go` currently hardcodes `":8080"` (all interfaces) and the model alias list `["opus","sonnet","haiku"]`. There is no way to change these without recompiling. The server is a local proxy intended to run on the user's workstation, so binding to loopback by default is both safer and more appropriate.

## Goals / Non-Goals

**Goals:**
- Default listen address becomes `127.0.0.1:8080`.
- Config file loaded from standard locations before startup.
- Config controls: `listen` address, model `aliases` list.
- `--config` CLI flag overrides the search path.
- Zero-config operation preserved: if no file is found, built-in defaults apply.

**Non-Goals:**
- Hot reload of config (restart required).
- Environment-variable-based configuration.
- TLS, authentication, or other server hardening.
- Validation beyond what Go's YAML parser rejects.

## Decisions

### Config format: YAML

**Chosen:** YAML via `gopkg.in/yaml.v3` (stdlib-quality, widely used in Go ecosystem).  
**Alternatives considered:**
- TOML — also human-friendly, but YAML is more universally known and tooled.
- JSON — no comments, poor UX for hand-edited files.
- INI — non-standard, less tooling.

YAML supports comments, is familiar to most developers, and `yaml.v3` is a mature, well-maintained library.

### Config search order

1. Path from `--config` flag (if provided, no fallback — error if missing).
2. `/etc/claude-code-openai-server/config.yaml` (system-wide).
3. `~/.claude-code-openai-server.yaml` (user, dot-file in home dir).
4. Built-in defaults (no file required).

For simplicity, only the first found file is loaded (no merge). This keeps the implementation trivial while covering the primary use cases.

### Config struct

```yaml
listen: "127.0.0.1:8080"       # TCP address to bind
aliases:                        # model aliases to probe at startup
  - opus
  - sonnet
  - haiku
```

Mapped to:

```go
type Config struct {
    Listen  string   `yaml:"listen"`
    Aliases []string `yaml:"aliases"`
}
```

Default values are set in a `defaultConfig()` function; the loaded file only overrides present keys.

### Flag parsing: `flag` stdlib

No external CLI library. A single `--config` flag added to `main.go` via the `flag` package is sufficient.

## Risks / Trade-offs

- **YAML dependency** → adds one external module. Mitigation: `gopkg.in/yaml.v3` is mature and widely used.
- **First-found, no merge** → system config is silently ignored when a user config exists. Mitigation: documented clearly; acceptable for a single-user local proxy.
- **`~` expansion** → `os.UserHomeDir()` may fail in unusual environments. Mitigation: fall through to next candidate on any error.

## Migration Plan

No migration needed. Default behaviour changes (bind address), but users running the server on `0.0.0.0` can restore it with a one-line config file. No data or persisted state is affected.

## Open Questions

None.
