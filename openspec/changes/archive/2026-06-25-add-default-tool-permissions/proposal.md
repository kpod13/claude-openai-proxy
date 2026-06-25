## Why

In headless `--print` mode the `claude` CLI has no TTY, so any tool call that needs an interactive permission prompt hangs forever and the HTTP request never returns. The current safe default (empty `allowed_tools` / `disallowed_tools`, mode `default`) leaves an operator who configures nothing exposed to exactly those hangs. We want a useful, hang-free default that still grants the low-risk web tools.

## What Changes

- When `permission.allowed_tools` is empty in config, default it to `WebSearch` and `WebFetch`.
- When `permission.disallowed_tools` is empty in config, default it to the remaining permission-requiring tools: `Artifact`, `Bash`, `Edit`, `ExitPlanMode`, `Monitor`, `NotebookEdit`, `PowerShell`, `ShareOnboardingGuide`, `Skill`, `Workflow`, `Write`.
- Defaulting is applied **per-section, independently**: a non-empty list in config is taken verbatim and is never merged with the built-in default.
- Read-only / no-permission tools (`Read`, `Grep`, `Glob`, `LS`, `LSP`, `Task*`, etc.) are deliberately left out of the disallow list — they never prompt, so they never hang.
- `mode` keeps its existing `default` value; the defaults work because allowlisted tools are pre-approved and disallowed tools fail fast instead of blocking.
- README documents the new default behavior.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `permission-policy`: the safe-by-default behavior changes from "empty tool lists / nothing permitted" to "empty sections fall back to built-in tool defaults (allow `WebSearch`/`WebFetch`, disallow the other permission-requiring tools), applied per-section." Validation, CLI-flag mapping, and `mode` defaulting are unchanged.

## Impact

- `internal/config/config.go`: apply per-section tool defaults during config load/validate; defaults must themselves pass existing validation.
- `internal/config` tests: cover empty, non-empty, and mixed sections.
- `README.md`: document the new default `allowed_tools` / `disallowed_tools`.
- No change to `internal/proxy` CLI-flag mapping — it already forwards whatever lists the config holds.
