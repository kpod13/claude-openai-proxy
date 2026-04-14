### Requirement: Go module exists
The repository SHALL contain a valid `go.mod` file at the root with a declared module path and a minimum Go version of 1.22.

#### Scenario: Module is valid
- **WHEN** `go mod verify` is run at the repository root
- **THEN** the command exits with code 0 and reports no errors

### Requirement: Server entry point
The project SHALL provide a `main` package under `cmd/server/main.go` that starts an HTTP server on a configurable port (default 8080).

#### Scenario: Server starts and listens
- **WHEN** the compiled binary is executed without arguments
- **THEN** the process binds to port 8080 and accepts TCP connections

#### Scenario: Server responds to health check
- **WHEN** a GET request is sent to `/healthz`
- **THEN** the server responds with HTTP 200 and body `ok`

### Requirement: Standard project layout
The project SHALL follow the standard Go project layout with `cmd/` for entry points and `internal/` for private packages.

#### Scenario: Build succeeds from root
- **WHEN** `go build ./...` is run at the repository root
- **THEN** all packages compile without errors

### Requirement: Makefile provides dev targets
The project SHALL include a `Makefile` at the root with `build`, `run`, and `test` targets.

#### Scenario: Build target compiles the binary
- **WHEN** `make build` is executed
- **THEN** a binary is produced at `bin/server`

#### Scenario: Test target runs Go tests
- **WHEN** `make test` is executed
- **THEN** `go test ./...` is invoked and exits with code 0 when all tests pass
