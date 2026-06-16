## Context

The proxy invokes `claude` as a subprocess in `--print` (headless) mode from `internal/proxy/claude.go`. There is no TTY, so any tool call that needs approval blocks indefinitely and the OpenAI HTTP request hangs. Operators currently have no way to set a permission policy; the only knobs are the hard-coded flag lists in `RunBlocking`, `RunBlockingImages`, `RunStreaming`, and `RunStreamingImages`.

The proxy is a high-privilege surface: every OpenAI request becomes tool execution on the host. Whatever policy we expose must be safe by default and require explicit opt-in for anything broader.

## Goals / Non-Goals

**Goals:**
- Let operators configure a permission policy in the config file.
- Map that policy to `claude` CLI flags on every invocation path.
- Default to the safest behavior, identical to today's, with no tools newly permitted.
- Keep the four `Run*` functions consistent so all request shapes honor the same policy.

**Non-Goals:**
- Relaying per-request permission prompts back through the OpenAI API (impossible — the protocol has no such channel, and the caller is a program, not a human).
- Per-request or per-API-key policy overrides; policy is server-wide for this change.
- Sandboxing or OS-level isolation beyond what `claude`'s own flags provide.

## Decisions

**Config shape — a nested `permission` block.** Group the keys under one block rather than scattering top-level keys, mirroring the existing `rate_limit` block:

```yaml
permission:
  mode: default                 # acceptEdits | auto | bypassPermissions | default | dontAsk | plan
  allowed_tools: []             # e.g. ["Write", "Edit", "Bash(git *)"]
  disallowed_tools: []
  add_dirs: []                  # extra directories tools may access
```
The supported `mode` values are taken from the `claude` CLI `--permission-mode` flag choices, so the proxy never forwards a value the CLI would reject.
Alternative considered: top-level keys. Rejected for consistency with `rate_limit` and to keep the namespace tidy.

**Safest default.** When the `permission` block is absent or empty: `mode` defaults to `default` and all lists are empty. The flag builder emits no `--permission-mode` override beyond `default` and no allowlist, so headless behavior is exactly as it is today — tool calls that need approval are not silently granted. Enabling tool calls is a deliberate operator action (allowlist a tool, or choose `acceptEdits`/`bypassPermissions`). We do **not** default to `bypassPermissions`, which would be remote code execution for anyone who can reach the port.

**Flag assembly in one place.** Add a helper (e.g. `permissionArgs(cfg) []string`) that converts the policy into CLI args, and prepend its output to each `Run*` arg list. This keeps the four invocation paths identical and testable in isolation. The config is plumbed from the handler (which already holds `*config.Config`) into the `Run*` calls.

**Startup validation, fail fast.** The whole permission block is validated when config loads, before the listener binds. `mode` must be one of the `claude` CLI `--permission-mode` choices (`acceptEdits`, `auto`, `bypassPermissions`, `default`, `dontAsk`, `plan`). Tool entries must match the `claude` tool-spec shape — `<ToolName>` or `<ToolName>(<rule>)`, regex `^[A-Za-z][A-Za-z0-9_]*(\([^)]+\))?$` after trimming — which covers built-ins (`Bash`, `Edit`) and MCP tools (`mcp__server__tool`). `add_dirs` entries must be non-blank paths. Because each entry is passed verbatim as a single argv element to `claude`, **no entry may begin with `-`** — otherwise a config value could be smuggled in as a CLI flag (e.g. `--dangerously-skip-permissions`); this guard is enforced for all three lists. An invalid value is a startup error rather than a silent passthrough, since it would otherwise make every request fail at runtime. The mode set is sourced from the CLI so the proxy stays in sync with what `claude` actually accepts.

## Risks / Trade-offs

- [Operator sets `bypassPermissions` on a `0.0.0.0` bind] → Effectively unauthenticated RCE. Mitigation: document the risk prominently in README/CLAUDE.md and keep the default safe; recommend narrow `allowed_tools` + `add_dirs` scoped to a working directory.
- [`claude` CLI flag names change in a future version] → Invocations break. Mitigation: flags are centralized in one helper, so a rename is a one-line fix; covered by tests asserting the emitted args.
- [Allowlisted tools still hang if a request triggers a non-allowlisted tool] → Partial improvement only. Accepted: the policy is explicit; operators broaden it as needed. This is inherent to headless operation.

## Migration Plan

No migration needed. Existing configs without a `permission` block keep working unchanged (safe default). Rollback is removing the block / reverting the binary.

## Open Questions

None blocking. Per-key policy overrides could be a future change if needed.
