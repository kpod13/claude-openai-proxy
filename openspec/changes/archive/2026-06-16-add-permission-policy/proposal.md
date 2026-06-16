## Why

The proxy runs `claude` headless (`--print`), where there is no terminal to display or answer permission prompts. When a request causes Claude to use a tool that needs approval (e.g. `Write`, `Bash`), the subprocess blocks forever waiting for an answer that can never come, and the OpenAI request hangs. The OpenAI Chat Completions protocol has no channel to relay a permission request, so the policy must be decided server-side, ahead of time.

## What Changes

- Add an optional `permission` config block (`mode`, `allowed_tools`, `disallowed_tools`, `add_dirs`) to the server config, validated at startup (fail fast) with `mode` restricted to the `claude` CLI's supported `--permission-mode` values.
- Translate that config into `claude` CLI flags (`--permission-mode`, `--allowedTools`, `--disallowedTools`, `--add-dir`) on every invocation (blocking, streaming, image variants).
- Default to the **safest** policy: `mode: default` with no tools allowlisted, so tool use stays opt-in and nothing new is permitted unless the operator configures it. This preserves today's behavior by default while giving operators a way to make headless tool calls succeed instead of hang.
- Update `CLAUDE.md` so it describes this Go proxy project instead of the outdated OpenSpec-template content.

## Capabilities

### New Capabilities
- `permission-policy`: how the operator-configured permission policy maps to `claude` CLI flags so headless tool calls follow an explicit, safe-by-default policy instead of blocking on interactive prompts.

### Modified Capabilities
- `server-config`: adds the optional `permission` config block and its keys to the documented config format.
- `chat-completions`: the `claude` invocation now carries permission flags derived from config.

## Impact

- Code: `internal/config/config.go` (new fields + defaults), `internal/proxy/claude.go` (flag assembly for all `claude` invocations), `internal/proxy/handler.go` (plumb config through to invocations).
- Docs: `README.md` (configuration section), `CLAUDE.md` (full rewrite to match the actual project).
- Behavior: no change by default (safest policy); operators opt in to broader tool access.
- Dependency: relies on `claude` CLI flags `--permission-mode`, `--allowedTools`, `--disallowedTools`, `--add-dir`.
