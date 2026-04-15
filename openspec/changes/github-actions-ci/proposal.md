## Why

The repository has tests and a linter configured but no automated CI to run them on every push or pull request. Without CI, regressions and lint violations can merge undetected.

## What Changes

- Add a GitHub Actions workflow that runs on every push and pull request to `master`.
- Workflow jobs: lint (golangci-lint), then test with coverage check (minimum threshold enforced).
- Lint runs before tests; tests are skipped if lint fails.
- Add `make ci` target that runs the full workflow locally in Docker via `act`.

## Capabilities

### New Capabilities

- `ci`: GitHub Actions workflow enforcing lint, test, and minimum code coverage on every push/PR, plus a `make ci` target for local execution via `act` (Docker).

### Modified Capabilities

- `go-server`: `Makefile` gains a `ci` target (and a `.PHONY` entry).

## Impact

- New file: `.github/workflows/ci.yml`.
- Modified file: `Makefile` — new `ci` target calling `act`.
- No changes to application code or existing tooling configuration.
- Requires `GITHUB_TOKEN` (implicit in Actions) — no secrets needed.
- Local `make ci` requires Docker and `act` installed on the developer's machine.
