## ADDED Requirements

### Requirement: Permission policy config block
The config file SHALL support an optional `permission` block controlling the headless `claude` permission policy, with the following keys:

- `mode` (string, default `default`): one of the values supported by the `claude` CLI `--permission-mode` flag — `acceptEdits`, `auto`, `bypassPermissions`, `default`, `dontAsk`, `plan`.
- `allowed_tools` (array of strings, default empty): tool specs to allow without prompting. Each entry is `<ToolName>` or `<ToolName>(<rule>)`, e.g. `"Write"`, `"Bash(git *)"`, `"mcp__server__tool"`.
- `disallowed_tools` (array of strings, default empty): tool specs to deny, same format as `allowed_tools`.
- `add_dirs` (array of strings, default empty): additional directory paths tools may access (absolute paths recommended).

When the block is absent, the server SHALL use the safe default (mode `default`, all lists empty). The server SHALL validate every permission value at startup (see the `permission-policy` capability) and fail fast on an unrecognised `mode`, a malformed tool spec, a blank entry, or any entry beginning with `-`.

#### Scenario: Permission block parsed
- **WHEN** the config contains `permission: {mode: acceptEdits, allowed_tools: ["Write"]}`
- **THEN** the server applies mode `acceptEdits` and allowlists the `Write` tool on `claude` invocations

#### Scenario: Permission block absent
- **WHEN** the config contains no `permission` block
- **THEN** the server uses mode `default` with no allowlisted tools

#### Scenario: Invalid mode rejected
- **WHEN** the config sets `permission: {mode: yolo}`
- **THEN** the server exits at startup with an informative error
