## Purpose

Defines the optional permission policy that controls how the headless `claude` subprocess handles tool permissions: the policy model, its safe defaults, how it maps to `claude` CLI flags, and how it is validated at startup.
## Requirements
### Requirement: Permission policy configuration
The server SHALL accept an optional permission policy that controls how the headless `claude` subprocess handles tool permissions. The policy consists of a mode and three lists: allowed tools, disallowed tools, and additional accessible directories. When mode is `default` and a tool list is empty, the server SHALL substitute a built-in default for that list (see "Safe-by-default policy").

#### Scenario: Policy configured
- **WHEN** the config specifies a permission `mode` and/or tool lists
- **THEN** the server applies that policy to every `claude` invocation it makes

#### Scenario: Policy absent
- **WHEN** the config contains no permission policy
- **THEN** the server applies mode `default`, the default allowlist (`WebSearch`, `WebFetch`), and the default disallow list (the other permission-requiring tools)

### Requirement: Safe-by-default policy
The server SHALL default permission `mode` to `default` and SHALL NOT default to `bypassPermissions` or any mode that auto-approves tool use. Under mode `default` only, the server SHALL apply hang-free built-in tool-list defaults per-section, independently, so an operator who configures nothing still gets the low-risk web tools without headless tool calls hanging on an interactive prompt:

- When mode is `default` and `allowed_tools` is empty, the server SHALL default it to exactly `WebSearch` and `WebFetch`.
- When mode is `default` and `disallowed_tools` is empty, the server SHALL default it to exactly the remaining permission-requiring tools: `Artifact`, `Bash`, `Edit`, `ExitPlanMode`, `Monitor`, `NotebookEdit`, `PowerShell`, `ShareOnboardingGuide`, `Skill`, `Workflow`, `Write`.
- A non-empty list in config SHALL be used verbatim and SHALL NOT be merged with its built-in default.
- When mode is anything other than `default`, the server SHALL leave empty tool lists empty: a non-default mode is an explicit operator choice (e.g. `acceptEdits`, `bypassPermissions`), and injecting the deny-list defaults would silently override that intent.

Read-only / no-permission tools (e.g. `Read`, `Grep`, `Glob`, `LS`, `LSP`) are intentionally excluded from the default disallow list because they never prompt and therefore never hang.

#### Scenario: Default allowlist applied
- **WHEN** mode is `default` (configured or absent) and `allowed_tools` is empty or absent
- **THEN** the policy allowlists exactly `WebSearch` and `WebFetch`

#### Scenario: Default disallow list applied
- **WHEN** mode is `default` (configured or absent) and `disallowed_tools` is empty or absent
- **THEN** the policy disallows exactly `Artifact`, `Bash`, `Edit`, `ExitPlanMode`, `Monitor`, `NotebookEdit`, `PowerShell`, `ShareOnboardingGuide`, `Skill`, `Workflow`, and `Write`

#### Scenario: Configured list overrides its default
- **WHEN** `allowed_tools` is set to a non-empty list in config
- **THEN** the policy uses that list exactly and does not add the default `WebSearch`/`WebFetch`

#### Scenario: Per-section defaulting is independent
- **WHEN** mode is `default` and `allowed_tools` is set but `disallowed_tools` is empty
- **THEN** the policy uses the configured `allowed_tools` and the default disallow list

#### Scenario: Non-default mode skips tool defaults
- **WHEN** mode is set to a non-`default` value (e.g. `acceptEdits` or `bypassPermissions`) and the tool lists are empty
- **THEN** the policy leaves both tool lists empty and applies no built-in tool defaults

#### Scenario: Bypass mode never defaulted
- **WHEN** no permission mode is configured
- **THEN** the server uses mode `default` and never `bypassPermissions`

#### Scenario: Opt-in still possible for stricter or looser policy
- **WHEN** an operator wants a different tool set or mode
- **THEN** they can set `allowed_tools`, `disallowed_tools`, or `mode` in config to override the defaults

### Requirement: Policy to CLI flag mapping
The server SHALL translate the configured policy into `claude` CLI flags on every invocation path (blocking text, blocking image, streaming text, streaming image): `mode` maps to `--permission-mode`, `allowed_tools` to `--allowedTools`, `disallowed_tools` to `--disallowedTools`, and `add_dirs` to `--add-dir`. Empty lists and the default mode produce no corresponding flags.

#### Scenario: Tools allowlisted
- **WHEN** the policy lists `allowed_tools: ["Write", "Edit"]`
- **THEN** the invocation includes `--allowedTools Write Edit`

#### Scenario: Mode selected
- **WHEN** the policy sets `mode: acceptEdits`
- **THEN** the invocation includes `--permission-mode acceptEdits`

#### Scenario: Additional directory granted
- **WHEN** the policy sets `add_dirs: ["/srv/work"]`
- **THEN** the invocation includes `--add-dir /srv/work`

#### Scenario: Consistent across request shapes
- **WHEN** the same policy is configured
- **THEN** streaming, non-streaming, and image-bearing requests all carry the identical permission flags

### Requirement: Startup validation of permission config
The server SHALL validate the entire permission policy at startup, before serving any request, and SHALL fail fast with an informative, non-zero exit when any value is invalid. Each list entry is passed verbatim as a single `claude` argv element, so every entry MUST also be rejected if it begins with `-` (to prevent it being smuggled in as a CLI flag). Validation rules:

- `mode` MUST be one of the values supported by the `claude` CLI `--permission-mode` flag: `acceptEdits`, `auto`, `bypassPermissions`, `default`, `dontAsk`, `plan`. An empty/absent `mode` is treated as `default`.
- Each `allowed_tools` / `disallowed_tools` entry MUST match the `claude` tool-spec format: a tool name optionally followed by a parenthesized rule — `<ToolName>` or `<ToolName>(<rule>)` — where `<ToolName>` matches `^[A-Za-z][A-Za-z0-9_]*$` (covers built-ins such as `Bash`, `Edit`, `Write` and MCP tools such as `mcp__server__tool`) and `<rule>`, when present, is a non-empty parenthesized string, e.g. `Bash(git *)`, `WebFetch(domain:example.com)`. Entries MUST be trimmed of surrounding whitespace and MUST be non-empty.
- Each `add_dirs` entry MUST be a non-empty, non-blank filesystem path (absolute paths recommended) and MUST NOT begin with `-`.

#### Scenario: Unknown mode rejected
- **WHEN** the config sets `mode: yolo`
- **THEN** the server exits at startup with a non-zero code and an error naming the invalid mode and the supported values

#### Scenario: Every supported mode accepted
- **WHEN** the config sets `mode` to any of `acceptEdits`, `auto`, `bypassPermissions`, `default`, `dontAsk`, or `plan`
- **THEN** the server starts and applies that mode

#### Scenario: Well-formed tool spec accepted
- **WHEN** `allowed_tools` contains `Write`, `Edit`, `Bash(git *)`, or `mcp__github__create_issue`
- **THEN** the server accepts the policy and forwards those specs to `claude`

#### Scenario: Malformed tool spec rejected
- **WHEN** `allowed_tools` or `disallowed_tools` contains an entry that is not a valid tool spec (e.g. `Bash(` with an unclosed rule, `123tool`, or an empty/whitespace-only entry)
- **THEN** the server exits at startup with a non-zero code and an informative error naming the offending entry

#### Scenario: Flag-like entry rejected
- **WHEN** any `allowed_tools`, `disallowed_tools`, or `add_dirs` entry begins with `-` (e.g. `--dangerously-skip-permissions`)
- **THEN** the server exits at startup with a non-zero code and an informative error

#### Scenario: Blank add_dirs entry rejected
- **WHEN** `add_dirs` contains an empty or whitespace-only entry
- **THEN** the server exits at startup with a non-zero code and an informative error

#### Scenario: Validation runs before serving
- **WHEN** the permission config is invalid
- **THEN** the server does not bind the listener or accept any request

