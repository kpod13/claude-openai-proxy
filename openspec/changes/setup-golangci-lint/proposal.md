## Why

The codebase has no static analysis beyond `go vet`. Without a linter configuration, code quality issues, unsafe patterns, and deviations from modern Go idioms accumulate silently. golangci-lint v2 is already installed; this change wires it into the project with a curated strict configuration.

## What Changes

- Add `.golangci.yml` at the repo root with a strict, opinionated linter selection targeting modern Go (1.22+)
- Add a `lint` target to the `Makefile`
- Fix any violations the new config surfaces in existing code

## Capabilities

### New Capabilities

- `linting`: Static analysis configuration enforcing correctness, modern idioms, error-handling discipline, and style

### Modified Capabilities

- `go-server`: Existing source files may be updated to satisfy the new lint rules
