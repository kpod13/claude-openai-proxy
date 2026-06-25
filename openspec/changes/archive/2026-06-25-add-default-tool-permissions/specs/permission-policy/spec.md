## MODIFIED Requirements

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
