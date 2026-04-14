## 1. Go Module Setup

- [x] 1.1 Run `go mod init github.com/timur/claude-code-openai-server` at the repo root to create `go.mod`
- [x] 1.2 Verify `go mod verify` exits with code 0

## 2. Project Layout

- [x] 2.1 Create `cmd/server/` directory
- [x] 2.2 Create `internal/` directory (placeholder for future packages)

## 3. HTTP Server Implementation

- [x] 3.1 Write `cmd/server/main.go` with a `net/http` server listening on port 8080
- [x] 3.2 Register a `/healthz` handler that returns HTTP 200 with body `ok`
- [x] 3.3 Verify `go build ./...` succeeds from the repo root

## 4. Makefile

- [x] 4.1 Create `Makefile` at the repo root with a `build` target that outputs `bin/server`
- [x] 4.2 Add a `run` target that builds and executes the server
- [x] 4.3 Add a `test` target that runs `go test ./...`

## 5. Git Hygiene

- [x] 5.1 Add a `.gitignore` entry for `bin/` and Go build cache (`*.test`, `*.out`)
- [x] 5.2 Confirm no build artifacts are tracked by git
