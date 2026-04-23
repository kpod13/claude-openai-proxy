## ADDED Requirements

### Requirement: DebugMiddleware logs request headers
`DebugMiddleware` SHALL include request headers in the DEBUG log entry for each incoming request. The `Authorization` header value SHALL be masked, showing only the scheme (e.g. `Bearer ***`).

#### Scenario: Request headers logged in verbose mode
- **WHEN** the server is started with `--verbose` and an HTTP request with headers arrives
- **THEN** a DEBUG entry is written containing the request header names and values

#### Scenario: Authorization header is masked
- **WHEN** the incoming request contains an `Authorization` header
- **THEN** the logged value shows only the scheme followed by `***` (e.g. `Bearer ***`), never the actual token

#### Scenario: Request headers absent in default mode
- **WHEN** the server is started without `--verbose`
- **THEN** no request header information is written to the log

### Requirement: DebugMiddleware logs response headers
`DebugMiddleware` SHALL include response headers in the DEBUG log entry written after the handler responds.

#### Scenario: Response headers logged in verbose mode
- **WHEN** the server is started with `--verbose` and a handler sets response headers
- **THEN** a DEBUG entry is written containing the response header names and values

#### Scenario: Response headers absent in default mode
- **WHEN** the server is started without `--verbose`
- **THEN** no response header information is written to the log
