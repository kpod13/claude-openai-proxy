## ADDED Requirements

### Requirement: Release workflow file exists
The repository SHALL contain `.github/workflows/release.yml` defining a GitHub Actions release workflow.

#### Scenario: Workflow file present
- **WHEN** the repository is checked out
- **THEN** `.github/workflows/release.yml` exists and is valid YAML

### Requirement: Workflow triggers on version tag
The release workflow SHALL trigger only on push of tags matching the pattern `v*.*.*`.

#### Scenario: Version tag triggers workflow
- **WHEN** a tag matching `v*.*.*` is pushed to the repository
- **THEN** the release workflow starts automatically

#### Scenario: Non-tag push does not trigger workflow
- **WHEN** a commit is pushed to a branch (not a tag)
- **THEN** the release workflow does not start

### Requirement: Lint and test gate before build
The release workflow SHALL run lint and test jobs before the build matrix, and the build SHALL only proceed if both pass.

#### Scenario: Lint failure blocks release
- **WHEN** the lint job fails
- **THEN** the build and publish steps are skipped and the workflow is marked failed

#### Scenario: Test failure blocks release
- **WHEN** any test fails or coverage is below the threshold
- **THEN** the build and publish steps are skipped and the workflow is marked failed

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
- **THEN** one binary artifact exists for each of the eight platform/arch combinations

### Requirement: Version embedded and debug info stripped
Each binary SHALL be built with `-ldflags "-s -w -X main.version=<tag>"` where `<tag>` is the pushed git tag. The `-s` and `-w` flags strip the symbol table and DWARF debug info to minimize binary size.

#### Scenario: Binary reports correct version
- **WHEN** the binary is executed with the `--version` flag
- **THEN** it prints the tag that triggered the release

#### Scenario: Binary has no debug info
- **WHEN** the binary is inspected with `file` or `objdump`
- **THEN** it contains no DWARF debug sections

### Requirement: GitHub Release created with artifacts
The workflow SHALL create a GitHub Release for the pushed tag, attach all platform binaries as raw files, and include auto-generated release notes from git history. Binary names SHALL follow the pattern `claude-openai-proxy-<os>-<arch>` (with `.exe` suffix for Windows).

#### Scenario: Release created on successful build
- **WHEN** all platform builds succeed
- **THEN** a GitHub Release exists for the tag with all eight binaries attached

#### Scenario: Release notes generated
- **WHEN** the GitHub Release is created
- **THEN** the release body contains auto-generated notes based on commits since the previous tag
