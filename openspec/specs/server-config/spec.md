### Requirement: Config file locations
The server SHALL search for a config file in the following order, using the first one found:
1. Path provided via `--config` flag (if given; error if file not found).
2. `/etc/claude-code-openai-server/config.yaml`.
3. `~/.claude-code-openai-server.yaml`.
If no file is found, built-in defaults apply and startup proceeds normally.

#### Scenario: System config loaded
- **WHEN** `/etc/claude-code-openai-server/config.yaml` exists and no `--config` flag is given
- **THEN** the server reads that file and applies its settings

#### Scenario: User config takes precedence over system config
- **WHEN** both `/etc/claude-code-openai-server/config.yaml` and `~/.claude-code-openai-server.yaml` exist
- **THEN** the server loads only `~/.claude-code-openai-server.yaml`

#### Scenario: Explicit config path used
- **WHEN** `--config /path/to/custom.yaml` is passed and the file exists
- **THEN** the server loads that file, ignoring standard locations

#### Scenario: Explicit config path missing
- **WHEN** `--config /path/to/missing.yaml` is passed and the file does not exist
- **THEN** the server exits with a non-zero code and an informative error message

#### Scenario: No config file found
- **WHEN** none of the standard locations contain a config file and no `--config` flag is given
- **THEN** the server starts successfully using built-in defaults

### Requirement: Config file format
The config file SHALL use YAML format with the following optional keys:

- `listen` (string): TCP address to bind, e.g. `"127.0.0.1:8080"`.
- `aliases` (array of strings): model aliases to probe at startup, e.g. `["opus", "sonnet", "haiku"]`.
- `rate_limit` (object): optional rate limiting settings. When absent or when both sub-keys are 0, rate limiting is disabled.
  - `requests_per_minute` (integer, default 0): maximum number of requests per key per minute. 0 means unlimited.
  - `tokens_per_minute` (integer, default 0): maximum number of tokens (prompt tokens) per key per minute. 0 means unlimited.

Unrecognised keys SHALL be ignored.

#### Scenario: Valid config parsed
- **WHEN** the config file contains `listen = "0.0.0.0:9090"` and `aliases = ["sonnet"]`
- **THEN** the server binds to `0.0.0.0:9090` and probes only the `sonnet` alias

#### Scenario: Partial config merges with defaults
- **WHEN** the config file contains only `listen = "0.0.0.0:9090"`
- **THEN** the server uses `0.0.0.0:9090` for the address and the default alias list

#### Scenario: Rate limit config parsed
- **WHEN** the config file contains `rate_limit: {requests_per_minute: 60, tokens_per_minute: 10000}`
- **THEN** the server enforces those limits on incoming requests

#### Scenario: Rate limit config absent
- **WHEN** the config file contains no `rate_limit` block
- **THEN** the server starts with rate limiting disabled (no limits enforced)

### Requirement: Default listen address
The server SHALL bind to `127.0.0.1:8080` when no `listen` value is specified in the config file and no `--config` flag overrides it.

#### Scenario: Default bind address
- **WHEN** the server starts with no config file present
- **THEN** it listens on `127.0.0.1:8080`

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
