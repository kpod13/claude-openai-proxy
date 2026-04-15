## ADDED Requirements

### Requirement: Lint configuration exists
The repository SHALL contain a `.golangci.yml` file at the root that configures golangci-lint v2 with a curated strict linter set.

#### Scenario: Config is valid
- **WHEN** `golangci-lint config verify` is run at the repo root
- **THEN** the command exits with code 0

### Requirement: Lint passes on all source files
All Go source files in the repository SHALL pass the configured linter set with zero reported issues.

#### Scenario: Clean lint run
- **WHEN** `golangci-lint run ./...` is run at the repo root
- **THEN** the command exits with code 0 and reports no issues

### Requirement: Makefile lint target
The `Makefile` SHALL include a `lint` target that invokes `golangci-lint run ./...`.

#### Scenario: Lint target executes
- **WHEN** `make lint` is executed
- **THEN** golangci-lint runs against all packages and exits with code 0 on a clean codebase

### Requirement: Correctness linters enabled
The configuration SHALL enable linters that catch bugs and unsafe patterns: `errcheck`, `govet`, `staticcheck`, `unused`, `bodyclose`, `nilerr`, `nilnil`, `nilnesserr`, `errorlint`, `durationcheck`, `makezero`, `reassign`, `forcetypeassert`, `noctx`, `fatcontext`.

#### Scenario: Unchecked error detected
- **WHEN** a function return error is silently discarded
- **THEN** `errcheck` reports a violation

### Requirement: Modern idiom linters enabled
The configuration SHALL enable linters that enforce modern Go (1.22+) usage: `modernize`, `intrange`, `copyloopvar`, `exptostd`, `usestdlibvars`, `perfsprint`, `mirror`, `prealloc`.

#### Scenario: Old-style loop variable capture flagged
- **WHEN** a loop variable is unnecessarily copied inside a goroutine
- **THEN** `copyloopvar` reports the issue

### Requirement: Error handling linters enabled
The configuration SHALL enable `err113`, `wrapcheck`, and `errname` to enforce disciplined error creation, wrapping, and naming.

#### Scenario: Bare error creation flagged
- **WHEN** `errors.New` or `fmt.Errorf` is used without wrapping an underlying cause where one exists
- **THEN** `err113` or `wrapcheck` reports the violation

### Requirement: Security linter enabled
The configuration SHALL enable `gosec` to catch common security issues.

#### Scenario: Security issue detected
- **WHEN** code contains a known insecure pattern (e.g., subprocess with unsanitised input)
- **THEN** `gosec` reports the finding
