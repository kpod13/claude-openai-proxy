## MODIFIED Requirements

### Requirement: Cross-platform binary matrix
The workflow SHALL build binaries for the following platform combinations using `go build`:
- linux/amd64
- linux/arm64
- darwin/amd64
- darwin/arm64
- windows/amd64
- windows/arm64
- freebsd/amd64
- freebsd/arm64

#### Scenario: All platform binaries produced
- **WHEN** the build matrix completes successfully
- **THEN** one binary is produced for each of the eight platform/arch combinations

### Requirement: Version embedded and debug info stripped
Each binary SHALL be built with `-ldflags "-s -w -X main.version=<tag>"` where `<tag>` is the pushed git tag. The `-s` and `-w` flags strip the symbol table and DWARF debug info to minimize binary size.

#### Scenario: Binary reports correct version
- **WHEN** the binary is executed with the `--version` flag
- **THEN** it prints the tag that triggered the release

#### Scenario: Binary has no debug info
- **WHEN** the binary is inspected with `file` or `objdump`
- **THEN** it contains no DWARF debug sections

### Requirement: GitHub Release created with artifacts
The workflow SHALL create a GitHub Release for the pushed tag and attach, for each platform, a versioned archive (`.tar.gz` for Unix-like systems, `.zip` for Windows) together with a `checksums.txt` file covering all archives. The release SHALL include auto-generated release notes. Each archive SHALL contain the executable named `claude-openai-proxy` (with `.exe` suffix on Windows).

#### Scenario: Release created on successful build
- **WHEN** all platform builds succeed
- **THEN** a GitHub Release exists for the tag with one archive per platform plus a `checksums.txt`

#### Scenario: Release notes generated
- **WHEN** the GitHub Release is created
- **THEN** the release body contains auto-generated notes based on commits since the previous tag

#### Scenario: Checksums cover all archives
- **WHEN** `checksums.txt` is inspected
- **THEN** it lists a SHA-256 entry for every published archive

## ADDED Requirements

### Requirement: Homebrew formula published on release
On each tagged release, the workflow SHALL render and publish the Homebrew formula `Formula/claude-openai-proxy.rb` to the tap repository (`kpod13/homebrew-tap`), with the `version` and the per-platform `url` + `sha256` set to the just-published release archives. Publishing SHALL use a token with write access to the tap repository, distinct from the default workflow `GITHUB_TOKEN`.

#### Scenario: Formula updated automatically on tag
- **WHEN** a `v*.*.*` tag is pushed and the release succeeds
- **THEN** the tap repository's `Formula/claude-openai-proxy.rb` is committed/updated to reference the new release archives and their checksums

#### Scenario: Cross-repo push uses dedicated token
- **WHEN** the workflow pushes the formula to the tap repository
- **THEN** it authenticates with a token granting write access to `kpod13/homebrew-tap`, not the default `GITHUB_TOKEN`
