## ADDED Requirements

### Requirement: Startup log includes discovered model names
When models are discovered at startup, the log message SHALL include the full list of model IDs, not just the count.

#### Scenario: Model names appear in startup log
- **WHEN** model discovery completes and at least one model is found
- **THEN** the log message includes each discovered model ID

#### Scenario: Count still present alongside names
- **WHEN** model discovery completes
- **THEN** the log message includes both the count and the model ID list

### Requirement: Handler logs HTTP requests and responses at DEBUG level
`proxy.Handler` SHALL emit a DEBUG log entry for each incoming HTTP request and a corresponding entry when the response is sent. These entries SHALL be suppressed at INFO level.

#### Scenario: Request logged in verbose mode
- **WHEN** the server is started with `--verbose` and an HTTP request arrives
- **THEN** a DEBUG entry is written containing the HTTP method and request path

#### Scenario: Response logged with status and duration in verbose mode
- **WHEN** the server is started with `--verbose` and an HTTP response is sent
- **THEN** a DEBUG entry is written containing the HTTP status code and elapsed duration

#### Scenario: No debug output in default mode
- **WHEN** the server is started without `--verbose`
- **THEN** no DEBUG entries are written for HTTP requests or responses

### Requirement: Handler logs CLI invocations at DEBUG level
`proxy.Handler` SHALL emit DEBUG log entries before and after each Claude CLI invocation, capturing the model used and token usage. These entries SHALL be suppressed at INFO level.

#### Scenario: CLI invocation logged with model in verbose mode
- **WHEN** the server is started with `--verbose` and a chat completion request triggers a CLI call
- **THEN** a DEBUG entry is written containing the model ID

#### Scenario: CLI result logged with token counts in verbose mode
- **WHEN** the server is started with `--verbose` and a CLI call completes successfully
- **THEN** a DEBUG entry is written containing input and output token counts

#### Scenario: No CLI debug output in default mode
- **WHEN** the server is started without `--verbose`
- **THEN** no DEBUG entries are written for CLI invocations or results

### Requirement: Nil logger is safe (no-op)
If `proxy.Handler.Logger` is nil, all debug logging calls SHALL be skipped without panicking. This preserves backward compatibility for callers that construct `Handler` without a logger.

#### Scenario: Handler with nil logger handles request without panic
- **WHEN** a `proxy.Handler` is constructed with `Logger` unset (nil)
- **THEN** requests are handled successfully with no panic
