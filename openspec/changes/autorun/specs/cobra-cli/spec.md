## ADDED Requirements

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
