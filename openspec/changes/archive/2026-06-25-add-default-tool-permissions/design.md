## Context

`internal/config/config.go` loads YAML into `Config`, merging over `defaultConfig()`, then runs `(*Config).validate()`, which normalizes and validates `Permission` (mode + `allowed_tools` / `disallowed_tools` / `add_dirs`). `internal/proxy/claude.go:PermissionArgs` maps the resulting lists to `--allowedTools` / `--disallowedTools` flags, emitting a flag only when the corresponding list is non-empty.

Today `defaultConfig()` sets only `Permission{Mode: ModeDefault}`, leaving both tool lists empty, so a default install allowlists nothing and disallows nothing — exposing headless requests to permission-prompt hangs.

## Goals / Non-Goals

**Goals:**
- Empty `allowed_tools` defaults to `WebSearch`, `WebFetch`.
- Empty `disallowed_tools` defaults to the other 11 permission-requiring tools.
- Per-section, independent defaulting; a non-empty list is used verbatim.
- Defaults are ordinary tool specs that pass existing validation unchanged.

**Non-Goals:**
- Changing `mode` defaulting, `add_dirs`, or the CLI-flag mapping in `internal/proxy`.
- Disallowing read-only / no-permission tools.
- Any merge semantics between a configured list and its default.

## Decisions

- **Where:** apply defaults in `(*Config).validate()` (or a helper it calls) in `internal/config/config.go`, before `validateToolSpecs` runs on each list, so defaults are validated like any other entry and a single code path covers both file-loaded and built-in-default configs. Detect "empty" as `len(list) == 0` after YAML decode.
- **Constants:** define `defaultAllowedTools = []string{"WebSearch", "WebFetch"}` and `defaultDisallowedTools = []string{"Artifact", "Bash", "Edit", "ExitPlanMode", "Monitor", "NotebookEdit", "PowerShell", "ShareOnboardingGuide", "Skill", "Workflow", "Write"}` as package-level values. Return copies when substituting so the shared slices are never mutated by in-place trimming in `validateToolSpecs`.
- **Per-section:** substitute each list independently — fill `AllowedTools` if empty, fill `DisallowedTools` if empty, regardless of the other.
- **`defaultConfig()`:** leave it setting only `Mode`; the substitution in `validate()` is the single source of the tool defaults so it also applies when a config file is present but omits the sections.

## Risks / Trade-offs

- **Behavior change:** a default install now allows `WebSearch`/`WebFetch` and disallows the 11 tools, where before it allowed/disallowed nothing. This is the intended fix; documented in README and the issue. An operator wanting the old "nothing" behavior can set an explicit `mode`/lists.
- **Default list drift:** the disallow list is a fixed snapshot of the current permission-requiring tools. If the `claude` CLI adds new permission-requiring tools they won't be covered until the list is updated. Acceptable; the list is centralized in one constant.
