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
