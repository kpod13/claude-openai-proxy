### Requirement: README exists at repo root
The repository SHALL contain a `README.md` file at the root level.

#### Scenario: File present
- **WHEN** a user visits the repository root
- **THEN** `README.md` is visible and rendered by GitHub

### Requirement: README describes what the project does
The README SHALL include a short description explaining that the project is an OpenAI-compatible HTTP proxy backed by the Claude CLI.

#### Scenario: Project purpose clear
- **WHEN** a user reads the opening section
- **THEN** they understand the proxy translates `/v1/chat/completions` and `/v1/models` to Claude CLI calls

### Requirement: README covers installation
The README SHALL document how to install the binary, including both `go install` and downloading a pre-built release.

#### Scenario: Install via go install
- **WHEN** a user follows the install instructions
- **THEN** they can run `go install` and get a working binary

### Requirement: README covers configuration
The README SHALL document the YAML config file format, including all supported keys: `listen`, `aliases`, and `rate_limit` (with `requests_per_minute` and `tokens_per_minute`).

#### Scenario: Config file example present
- **WHEN** a user reads the configuration section
- **THEN** they see a complete YAML example with all optional fields

### Requirement: README covers rate limiting
The README SHALL explain that rate limiting is disabled by default and show how to enable it via the config file.

#### Scenario: Rate limiting section present
- **WHEN** a user reads the README
- **THEN** they find a section explaining RPM/TPM limits and the OpenAI-compatible headers returned

### Requirement: README includes development commands
The README SHALL list the available `make` targets: `build`, `run`, `test`, `lint`.

#### Scenario: Dev commands documented
- **WHEN** a contributor reads the README
- **THEN** they can find how to build, test, and lint the project
