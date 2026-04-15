## MODIFIED Requirements

### Requirement: Server entry point
The project SHALL provide a `main` package under `cmd/server/main.go` that starts an HTTP server on a configurable port (default 8080) and registers the following routes: `/healthz`, `GET /v1/models`, and `POST /v1/chat/completions`.

#### Scenario: Server starts and listens
- **WHEN** the compiled binary is executed without arguments
- **THEN** the process binds to port 8080 and accepts TCP connections

#### Scenario: Server responds to health check
- **WHEN** a GET request is sent to `/healthz`
- **THEN** the server responds with HTTP 200 and body `ok`

#### Scenario: Proxy routes registered
- **WHEN** the server starts successfully
- **THEN** `GET /v1/models` and `POST /v1/chat/completions` return non-404 responses
