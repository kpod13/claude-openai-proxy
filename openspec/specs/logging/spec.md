### Requirement: Logger supports info, debug, and error levels
The logger package SHALL support three log levels: INFO (default), DEBUG (verbose), and ERROR. Messages below the configured level SHALL be suppressed.

#### Scenario: INFO level suppresses debug messages
- **WHEN** the logger is configured at INFO level
- **THEN** Debug() calls produce no output

#### Scenario: DEBUG level emits all messages
- **WHEN** the logger is configured at DEBUG level
- **THEN** both Info() and Debug() calls produce output

#### Scenario: Error messages always emitted unless quiet
- **WHEN** the logger is not in quiet mode and Error() is called
- **THEN** the message is written regardless of whether level is INFO or DEBUG

### Requirement: Logger supports plain and JSON output formats
The logger SHALL support two output formats: `plain` (human-readable key=value) and `json` (JSON lines). The format SHALL be selected at construction time.

#### Scenario: Plain format output
- **WHEN** the logger is constructed with format `plain`
- **THEN** log lines are written in text/key=value format to stderr

#### Scenario: JSON format output
- **WHEN** the logger is constructed with format `json`
- **THEN** log lines are written as JSON objects to stderr

### Requirement: Quiet mode disables all log output
When quiet mode is enabled, the logger SHALL produce no output for any log level, including errors.

#### Scenario: Quiet mode suppresses all output
- **WHEN** the logger is constructed with quiet mode enabled
- **THEN** no output is written for Info(), Debug(), or Error() calls

### Requirement: Quiet mode takes precedence over verbose
If both quiet and verbose are requested, quiet SHALL win and no output SHALL be produced.

#### Scenario: Quiet overrides verbose
- **WHEN** the logger is constructed with both quiet=true and level=DEBUG
- **THEN** no output is produced

### Requirement: Startup log includes discovered model names
When models are discovered at startup, the log message SHALL include the full list of model IDs, not just the count.

#### Scenario: Model names appear in startup log
- **WHEN** model discovery completes and at least one model is found
- **THEN** the log message includes each discovered model ID

#### Scenario: Count still present alongside names
- **WHEN** model discovery completes
- **THEN** the log message includes both the count and the model ID list

### Requirement: HTTP requests and responses are logged at DEBUG level via middleware
`DebugMiddleware` SHALL emit a DEBUG log entry for each incoming HTTP request and a corresponding entry when the response is sent. The middleware SHALL be applied only when the server is started with `--verbose`. These entries SHALL be suppressed at INFO level.

#### Scenario: Request logged in verbose mode
- **WHEN** the server is started with `--verbose` and an HTTP request arrives
- **THEN** a DEBUG entry is written containing the HTTP method and request path

#### Scenario: Response logged with status and duration in verbose mode
- **WHEN** the server is started with `--verbose` and an HTTP response is sent
- **THEN** a DEBUG entry is written containing the HTTP status code and elapsed duration

#### Scenario: No debug output in default mode
- **WHEN** the server is started without `--verbose`
- **THEN** no DEBUG entries are written for HTTP requests or responses

### Requirement: CLI invocations are logged at DEBUG level via wrappers
`DebugRunBlocking` and `DebugRunStreaming` SHALL emit DEBUG log entries before and after each Claude CLI invocation, capturing the model used and token usage. These wrappers SHALL be applied only when the server is started with `--verbose`. These entries SHALL be suppressed at INFO level.

#### Scenario: CLI invocation logged with model in verbose mode
- **WHEN** the server is started with `--verbose` and a chat completion request triggers a CLI call
- **THEN** a DEBUG entry is written containing the model ID

#### Scenario: CLI result logged with token counts in verbose mode
- **WHEN** the server is started with `--verbose` and a CLI call completes successfully
- **THEN** a DEBUG entry is written containing input and output token counts

#### Scenario: No CLI debug output in default mode
- **WHEN** the server is started without `--verbose`
- **THEN** no DEBUG entries are written for CLI invocations or results
