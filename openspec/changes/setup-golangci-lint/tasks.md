## 1. Configuration

- [ ] 1.1 Create `.golangci.yml` at the repo root with the full linter selection and settings as specified in design.md
- [ ] 1.2 Run `golangci-lint config verify` — must exit 0

## 2. Makefile

- [ ] 2.1 Add `lint` target to `Makefile` that runs `golangci-lint run ./...`

## 3. Fix lint violations

- [ ] 3.1 Run `golangci-lint run ./...` and collect all reported issues
- [ ] 3.2 Fix all reported violations in `internal/proxy/` and `cmd/server/`
- [ ] 3.3 Re-run `golangci-lint run ./...` — must exit 0 with no issues

## 4. Verification

- [ ] 4.1 Run `make lint` — exits 0
- [ ] 4.2 Run `go test ./...` — all tests still pass after any code changes
