### Requirement: Root command starts the server
The binary SHALL use a Cobra root command as its entry point. When invoked without a subcommand, it SHALL start the HTTP server using the loaded configuration.

#### Scenario: Server starts without subcommand
- **WHEN** the binary is executed with no subcommand
- **THEN** the server starts and listens on the configured address

#### Scenario: Config flag is accepted
- **WHEN** the binary is executed with `--config <path>`
- **THEN** configuration is loaded from the specified file

### Requirement: Version flag prints version and exits
The root command SHALL support a `--version` flag. When provided, it SHALL print the version string and exit with code 0 without starting the server.

#### Scenario: Version flag output
- **WHEN** the binary is executed with `--version`
- **THEN** the version string is printed to stdout and the process exits with code 0

### Requirement: Help output follows Cobra conventions
The root command SHALL print usage and flag descriptions when invoked with `--help` or `-h`.

#### Scenario: Help flag
- **WHEN** the binary is executed with `--help`
- **THEN** usage text including all flags and available subcommands is printed to stdout

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

### Requirement: autorun subcommand group
The CLI SHALL expose an `autorun` subcommand group with two sub-subcommands: `install` and `uninstall`.

#### Scenario: autorun help is available
- **WHEN** the binary is executed with `autorun --help`
- **THEN** usage text listing `install` and `uninstall` subcommands is printed to stdout

### Requirement: autorun install subcommand
The `autorun install` command SHALL provision user-level autostart for the current OS and print a confirmation message including the path of the autostart entry that was created.

#### Scenario: install prints confirmation
- **WHEN** `autorun install` completes successfully
- **THEN** a success message is printed to stdout including the path of the created entry

#### Scenario: install fails gracefully on unsupported OS
- **WHEN** `autorun install` is run on an unsupported OS
- **THEN** the command exits with a non-zero code and an informative error message

### Requirement: autorun uninstall subcommand
The `autorun uninstall` command SHALL remove the autostart entry for the current OS and print a confirmation message.

#### Scenario: uninstall prints confirmation
- **WHEN** `autorun uninstall` completes successfully
- **THEN** a success message is printed to stdout

#### Scenario: uninstall on already-clean system exits cleanly
- **WHEN** `autorun uninstall` is run and no entry exists
- **THEN** the command exits with code 0 and prints a message indicating nothing to remove
