## Purpose

Defines the optional permission policy that controls how the headless `claude` subprocess handles tool permissions: the policy model, its safe defaults, how it maps to `claude` CLI flags, and how it is validated at startup.
## Requirements
### Requirement: Permission policy configuration
The server SHALL accept an optional permission policy that controls how the headless `claude` subprocess handles tool permissions. The policy consists of a mode and three lists: allowed tools, disallowed tools, and additional accessible directories.

#### Scenario: Policy configured
- **WHEN** the config specifies a permission `mode` and/or tool lists
- **THEN** the server applies that policy to every `claude` invocation it makes

#### Scenario: Policy absent
- **WHEN** the config contains no permission policy
- **THEN** the server applies the safe default policy (mode `default`, no allowlisted tools)

### Requirement: Safe-by-default policy
The server SHALL default to the safest permission policy when none is configured: permission mode `default` and empty tool/directory lists, so no tool is newly permitted and headless behavior is unchanged from a build without this feature. The server SHALL NOT default to `bypassPermissions` or any mode that auto-approves tool use.

#### Scenario: Default mode emitted
- **WHEN** no permission policy is configured
- **THEN** the server does not emit an allowlist and does not bypass permission checks

#### Scenario: Opt-in required for tool access
- **WHEN** an operator wants headless tool calls to succeed instead of hang
- **THEN** they must explicitly allowlist tools or select a less strict mode in config

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
