## Context

The project has `golangci-lint` configured via `.golangci.yml` and tests in `internal/config/`. There is no CI pipeline. The goal is a minimal, fast workflow triggered on every push and PR to `master`.

## Goals / Non-Goals

**Goals:**
- Lint runs first; test job is skipped if lint fails (fast feedback, saves runner minutes).
- Tests run with coverage profiling; workflow fails if total coverage is below a minimum threshold.
- No external secrets required — only the implicit `GITHUB_TOKEN`.
- Single workflow file: `.github/workflows/ci.yml`.

**Non-Goals:**
- Build artifact publication or Docker image builds.
- Deployment or release automation.
- Matrix testing across multiple Go versions (pin to latest stable).
- Coverage reporting to external services (Codecov, Coveralls).

## Decisions

### Lint action: `golangci/golangci-lint-action`

**Chosen:** The official `golangci/golangci-lint-action`. It caches the linter binary and lint results, reads `.golangci.yml` automatically, and is the de-facto standard.  
**Alternative:** `go run github.com/golangci/golangci-lint/cmd/golangci-lint` — no caching, slower.

### Coverage enforcement: inline shell, no external action

**Chosen:** `go test -coverprofile=coverage.out ./...` followed by a shell snippet that extracts total coverage via `go tool cover -func` and compares against the threshold.  
**Alternative:** `vladopajic/go-test-coverage` action — adds an external dependency for trivial functionality.  
**Rationale:** Keeps the workflow self-contained; the logic is 3 lines of shell.

### Minimum coverage threshold: 70%

Starting at 70%. Easy to raise in `.github/workflows/ci.yml` as coverage improves. If current coverage is below 70% after the workflow is added, the threshold is lowered to match current coverage and a TODO is added.

### Job dependency: `test` needs `lint`

`needs: lint` ensures the test job only runs when lint is green. This avoids wasting runner time running tests against code that already has known issues.

### Go version: pinned to `1.24`

Matches the version used locally. Pinning avoids surprise breakage from minor-version behaviour changes; bumping is a deliberate action.

## Risks / Trade-offs

- **Coverage threshold too high** → workflow fails immediately after introduction. Mitigation: check actual coverage during task implementation and set threshold accordingly.
- **golangci-lint-action version drift** → use a pinned major version tag (e.g. `v6`) to avoid breaking changes from minor updates.

### Local CI: `act`

**Chosen:** [`act`](https://github.com/nektos/act) — runs GitHub Actions workflows locally inside Docker containers. The `make ci` target invokes `act push` to simulate a push-to-master event.  
**Alternative:** Manually running `make lint && make test` — does not replicate the exact Actions environment (container, env vars, job ordering).  
**Rationale:** `act` uses the same workflow YAML, so local and remote execution are identical by construction. Developers can catch failures before pushing.

### `act` image: `catthehacker/ubuntu:act-latest`

The default `act` micro image lacks Go. `catthehacker/ubuntu:act-latest` is the standard full image that includes common runtimes and matches the `ubuntu-latest` runner used in the workflow.  
The image is specified via `-P ubuntu-latest=catthehacker/ubuntu:act-latest` in the `make ci` command so no per-developer `.actrc` is required.

### `make ci` command

```makefile
ci:
    act push -P ubuntu-latest=catthehacker/ubuntu:act-latest
```

No `--secret` flags needed — the workflow uses only `GITHUB_TOKEN` which `act` provides automatically.

## Migration Plan

No existing CI to migrate. Add `.github/workflows/ci.yml` and verify the workflow runs green on the first push. Run `make ci` locally to validate before pushing.
