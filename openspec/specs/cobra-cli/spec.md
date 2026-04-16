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
