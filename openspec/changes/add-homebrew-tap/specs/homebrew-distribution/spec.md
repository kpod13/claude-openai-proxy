## ADDED Requirements

### Requirement: Homebrew tap repository exists
A dedicated GitHub repository SHALL serve as the Homebrew tap. It MUST be named with the `homebrew-` prefix (e.g. `kpod13/homebrew-tap`) so users can tap it using the short form, and it MUST contain a `Formula/` directory holding the formula file.

#### Scenario: Tap repo is named and structured correctly
- **WHEN** the tap repository is inspected
- **THEN** its name begins with `homebrew-` and it contains `Formula/claude-openai-proxy.rb`

#### Scenario: User taps using the short form
- **WHEN** a user runs `brew tap kpod13/tap`
- **THEN** Homebrew resolves it to the `kpod13/homebrew-tap` repository without error

### Requirement: Formula installs the proxy binary
The formula `Formula/claude-openai-proxy.rb` SHALL define a download `url` to a released archive, a matching `sha256`, the release `version`, and install the `claude-openai-proxy` executable onto the user's `PATH` via `bin.install`.

#### Scenario: Install places binary on PATH
- **WHEN** a user runs `brew install kpod13/tap/claude-openai-proxy`
- **THEN** the `claude-openai-proxy` command is available on `PATH`

#### Scenario: Download integrity is verified
- **WHEN** Homebrew downloads the release archive referenced by the formula
- **THEN** the archive's checksum matches the formula's `sha256` and a mismatch aborts installation

### Requirement: Formula includes a test block
The formula SHALL include a `test do` block that exercises the installed binary so `brew test` and Homebrew audits can verify it runs.

#### Scenario: brew test passes for a healthy build
- **WHEN** `brew test claude-openai-proxy` is run after install
- **THEN** the test block invokes the binary (e.g. its version/help command) and exits successfully

### Requirement: Users can upgrade via Homebrew
Once installed from the tap, the proxy SHALL be upgradeable through standard Homebrew commands when a newer formula version is published.

#### Scenario: Upgrade to a newer release
- **WHEN** a newer formula version is present in the tap and the user runs `brew upgrade claude-openai-proxy`
- **THEN** Homebrew installs the newer version, replacing the previous one

### Requirement: README documents Homebrew installation
The project README SHALL document the Homebrew install path, including the `brew tap` and `brew install` commands.

#### Scenario: Install instructions present
- **WHEN** a reader opens the README
- **THEN** it contains the `brew tap kpod13/tap` and `brew install claude-openai-proxy` commands
