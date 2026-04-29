### Requirement: Go module exists
The repository SHALL contain a valid `go.mod` file at the root with a declared module path and a minimum Go version of 1.22.

#### Scenario: Module is valid
- **WHEN** `go mod verify` is run at the repository root
- **THEN** the command exits with code 0 and reports no errors

### Requirement: Server entry point
The project SHALL provide a `main` package under `cmd/claude-openai-proxy/main.go` that starts an HTTP server on a configurable address (default `127.0.0.1:8080`) and registers the following routes: `/healthz`, `GET /v1/models`, and `POST /v1/chat/completions`. The listen address and model aliases SHALL be read from the config file (see `server-config` capability) before the server binds.

#### Scenario: Server starts and listens on loopback by default
- **WHEN** the compiled binary is executed without arguments and no config file is present
- **THEN** the process binds to `127.0.0.1:8080` and accepts TCP connections

#### Scenario: Server responds to health check
- **WHEN** a GET request is sent to `/healthz`
- **THEN** the server responds with HTTP 200 and body `ok`

#### Scenario: Proxy routes registered
- **WHEN** the server starts successfully
- **THEN** `GET /v1/models` and `POST /v1/chat/completions` return non-404 responses

### Requirement: Standard project layout
The project SHALL follow the standard Go project layout with `cmd/` for entry points and `internal/` for private packages.

#### Scenario: Build succeeds from root
- **WHEN** `go build ./...` is run at the repository root
- **THEN** all packages compile without errors

### Requirement: Makefile provides dev targets
The project SHALL include a `Makefile` at the root with `build`, `run`, `test`, `lint`, and `ci` targets.

#### Scenario: Build target compiles the binary
- **WHEN** `make build` is executed
- **THEN** a binary is produced at `bin/claude-openai-proxy`

#### Scenario: Test target runs Go tests
- **WHEN** `make test` is executed
- **THEN** `go test ./...` is invoked and exits with code 0 when all tests pass

#### Scenario: CI target runs the full workflow locally
- **WHEN** `make ci` is executed on a machine with Docker and `act` installed
- **THEN** `act` runs the GitHub Actions workflow in Docker and exits with code 0 when all jobs pass
