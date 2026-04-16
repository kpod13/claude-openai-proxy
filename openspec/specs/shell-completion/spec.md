### Requirement: Completion subcommand generates shell scripts
The binary SHALL expose a `completion <shell>` subcommand that prints an autocompletion script for the requested shell to stdout. Supported shells: `bash`, `zsh`, `fish`, `powershell`.

#### Scenario: Bash completion script
- **WHEN** the binary is executed with `completion bash`
- **THEN** a valid bash autocompletion script is printed to stdout

#### Scenario: Zsh completion script
- **WHEN** the binary is executed with `completion zsh`
- **THEN** a valid zsh autocompletion script is printed to stdout

#### Scenario: Fish completion script
- **WHEN** the binary is executed with `completion fish`
- **THEN** a valid fish autocompletion script is printed to stdout

#### Scenario: PowerShell completion script
- **WHEN** the binary is executed with `completion powershell`
- **THEN** a valid PowerShell autocompletion script is printed to stdout

#### Scenario: Unknown shell argument
- **WHEN** the binary is executed with `completion <unknown>`
- **THEN** an error message is printed and the process exits with a non-zero code

### Requirement: Completion subcommand has install instructions in help
The `completion` subcommand help text SHALL include brief per-shell instructions for how to install the generated script.

#### Scenario: Help for completion subcommand
- **WHEN** the binary is executed with `completion --help`
- **THEN** help text is printed including installation instructions for each supported shell
