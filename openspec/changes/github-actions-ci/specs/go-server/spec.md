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
