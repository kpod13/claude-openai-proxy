## MODIFIED Requirements

### Requirement: CI workflow file exists
The repository SHALL contain `.github/workflows/ci.yml` defining a GitHub Actions CI workflow, and `.github/workflows/release.yml` defining a release workflow.

#### Scenario: CI workflow file present
- **WHEN** the repository is checked out
- **THEN** `.github/workflows/ci.yml` exists and is valid YAML

#### Scenario: Release workflow file present
- **WHEN** the repository is checked out
- **THEN** `.github/workflows/release.yml` exists and is valid YAML
