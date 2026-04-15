## Why

The server currently hardcodes `0.0.0.0:8080`, exposing it on all interfaces with no way to change the address or port without recompiling. A config file allows operators to tune the server without touching the binary.

## What Changes

- Server binds to `127.0.0.1:8080` by default instead of `0.0.0.0:8080`.
- A TOML/YAML config file is loaded at startup from standard locations.
- Config file controls listen address, port, and discovered model aliases.
- CLI flag `--config` allows overriding the config file path.

## Capabilities

### New Capabilities

- `server-config`: Config file loading with XDG-style fallback — `/etc/claude-code-openai-server/config.toml`, then `~/.claude-code-openai-server.toml`; merged with built-in defaults.

### Modified Capabilities

- `go-server`: Default listen address changes from `0.0.0.0:8080` to `127.0.0.1:8080`; server startup now reads config before binding.

## Impact

- `cmd/server/main.go`: reads config, passes settings to handler/server setup.
- No changes to proxy handler or claude CLI invocation logic.
- New dependency: a config parsing library (TOML) or use stdlib `encoding/json` / a minimal TOML parser.
