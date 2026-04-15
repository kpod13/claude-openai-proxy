## MODIFIED Requirements

### Requirement: Makefile provides dev targets
The project SHALL include a `Makefile` at the root with `build`, `run`, `test`, `lint`, and `ci` targets.

#### Scenario: Build target compiles the binary
- **WHEN** `make build` is executed
- **THEN** a binary is produced at `bin/server`

#### Scenario: Test target runs Go tests
- **WHEN** `make test` is executed
- **THEN** `go test ./...` is invoked and exits with code 0 when all tests pass

#### Scenario: CI target runs the full workflow locally
- **WHEN** `make ci` is executed on a machine with Docker and `act` installed
- **THEN** `act` runs the GitHub Actions workflow in Docker and exits with code 0 when all jobs pass

## ADDED Requirements

### Requirement: CI workflow file exists
The repository SHALL contain `.github/workflows/ci.yml` defining a GitHub Actions workflow.

#### Scenario: Workflow file present
- **WHEN** the repository is checked out
- **THEN** `.github/workflows/ci.yml` exists and is valid YAML

### Requirement: Workflow triggers
The CI workflow SHALL trigger on every push to `master` and on every pull request targeting `master`.

#### Scenario: Push to master triggers workflow
- **WHEN** a commit is pushed to the `master` branch
- **THEN** the CI workflow starts automatically

#### Scenario: Pull request triggers workflow
- **WHEN** a pull request targeting `master` is opened or updated
- **THEN** the CI workflow starts automatically

### Requirement: Lint job
The workflow SHALL include a `lint` job that runs `golangci-lint` using the repository's `.golangci.yml` configuration.

#### Scenario: Lint passes on clean code
- **WHEN** all source files satisfy the configured linter rules
- **THEN** the lint job exits with code 0

#### Scenario: Lint fails on violation
- **WHEN** a source file contains a lint violation
- **THEN** the lint job exits with a non-zero code and the workflow is marked failed

### Requirement: Test job depends on lint
The `test` job SHALL declare `needs: lint` so it only runs when the lint job succeeds.

#### Scenario: Tests skipped on lint failure
- **WHEN** the lint job fails
- **THEN** the test job is skipped and does not consume runner minutes

### Requirement: Tests run with coverage
The `test` job SHALL run `go test -coverprofile=coverage.out ./...` and report total coverage.

#### Scenario: Tests pass
- **WHEN** all tests pass
- **THEN** the test job exits with code 0

#### Scenario: Test failure fails workflow
- **WHEN** any test fails
- **THEN** the test job exits with a non-zero code and the workflow is marked failed

### Requirement: Minimum coverage threshold
The `test` job SHALL fail if total code coverage is below the configured minimum threshold (default 70%).

#### Scenario: Coverage above threshold
- **WHEN** total coverage meets or exceeds the minimum threshold
- **THEN** the coverage check passes and the test job succeeds

#### Scenario: Coverage below threshold
- **WHEN** total coverage is below the minimum threshold
- **THEN** the test job exits with a non-zero code and prints the actual and required coverage
