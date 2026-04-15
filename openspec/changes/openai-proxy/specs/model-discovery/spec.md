## ADDED Requirements

### Requirement: Model discovery at startup
The server SHALL probe the `claude` CLI at startup using known aliases (`opus`, `sonnet`, `haiku`) to resolve full model IDs and build an in-memory model registry. Probes SHALL run concurrently.

#### Scenario: Successful probe
- **WHEN** the server starts and `claude` is available on PATH
- **THEN** each alias is resolved to a full model ID (e.g., `sonnet` → `claude-sonnet-4-6`) and stored in the registry

#### Scenario: CLI not on PATH
- **WHEN** the server starts and `claude` cannot be found
- **THEN** the server logs a fatal error and exits with a non-zero code

#### Scenario: Partial probe failure
- **WHEN** one alias probe fails (e.g., model temporarily unavailable)
- **THEN** that alias is omitted from the registry and the server continues with the remaining models

### Requirement: Model registry lookup
The server SHALL support looking up a model by full ID or alias. An unknown model identifier SHALL return an error.

#### Scenario: Lookup by full ID
- **WHEN** a request specifies `claude-sonnet-4-6`
- **THEN** the registry returns that model ID directly

#### Scenario: Lookup by alias
- **WHEN** a request specifies `sonnet`
- **THEN** the registry resolves it to the full model ID discovered at startup

#### Scenario: Unknown model
- **WHEN** a request specifies an unrecognized model name
- **THEN** the handler returns HTTP 400 with a descriptive error message
