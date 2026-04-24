## Why

The test suite has 107 test functions across 9 files. Many of these are near-identical functions that differ only in input and expected output, making them hard to maintain and extend. Table-driven tests with `t.Run()` reduce duplication, improve readability, and make it easy to add new cases.

## What Changes

- Consolidate repetitive `TestFoo_CaseA`, `TestFoo_CaseB` functions into single `TestFoo` functions using `cases := []struct{ ... }` + `t.Run()`
- Target files: `models_test.go`, `claude_test.go`, `logger_test.go`, `debug_test.go`, `handler_test.go`, `config_test.go`
- No functional changes — behavior, coverage, and test names (via subtests) remain equivalent

## Capabilities

### New Capabilities
- None

### Modified Capabilities
- None

## Impact

- Test files only — no production code changes
- Test count drops from ~107 functions to ~60-70 (exact count depends on groupings)
- CI and coverage gates unaffected
