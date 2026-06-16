## ADDED Requirements

### Requirement: Permission flags on claude invocation
The server SHALL include the configured permission policy flags (`--permission-mode`, `--allowedTools`, `--disallowedTools`, `--add-dir`) when invoking `claude` for `/v1/chat/completions`, for both streaming and non-streaming requests and for both text-only and image-bearing requests. With the safe default policy, no permission flags beyond the implicit `default` mode are added, leaving the invocation unchanged from prior behavior.

#### Scenario: Allowlisted tool call succeeds headlessly
- **WHEN** the operator has configured `allowed_tools` covering the tool a request triggers
- **THEN** the `claude` invocation carries the allowlist and the tool runs without blocking on an interactive prompt

#### Scenario: Default policy leaves invocation unchanged
- **WHEN** no permission policy is configured
- **THEN** the `claude` invocation contains no allowlist and no permission-bypass flags
