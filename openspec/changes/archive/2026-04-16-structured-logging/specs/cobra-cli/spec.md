## ADDED Requirements

### Requirement: Root command accepts --verbose flag
The root command SHALL accept a `--verbose` flag. When set, the logger SHALL operate at DEBUG level, emitting detailed output.

#### Scenario: Verbose flag enables debug output
- **WHEN** the binary is executed with `--verbose`
- **THEN** debug-level log messages are written to stderr

### Requirement: Root command accepts --quiet flag
The root command SHALL accept a `--quiet` flag. When set, all log output SHALL be suppressed regardless of other flags.

#### Scenario: Quiet flag suppresses all output
- **WHEN** the binary is executed with `--quiet`
- **THEN** no log output is written to stderr

### Requirement: Root command accepts --log-format flag
The root command SHALL accept a `--log-format` flag with values `plain` (default) or `json`. The selected format SHALL be applied to all log output.

#### Scenario: Default log format is plain
- **WHEN** the binary is executed without `--log-format`
- **THEN** log output is in human-readable text format

#### Scenario: JSON log format
- **WHEN** the binary is executed with `--log-format=json`
- **THEN** log output is written as JSON lines
