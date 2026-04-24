## Context

The test suite has grown to 107 test functions. Many follow the pattern `TestFoo_CaseA` / `TestFoo_CaseB` where each function tests the same code path with different inputs. Go's idiomatic approach is to unify these into a single `TestFoo` with a `cases` slice and `t.Run()` subtests.

## Goals / Non-Goals

**Goals:**
- Reduce test function count by consolidating repetitive variants into table-driven tests
- Maintain equivalent coverage and subtest name readability
- Keep each test file self-contained and easy to extend

**Non-Goals:**
- Changing test behavior or coverage
- Modifying production code
- Introducing test helpers or shared fixtures beyond what already exists

## Decisions

**Use anonymous struct slices, not named types.** Each table is local to its test function; naming the struct adds no reuse value.

**Preserve descriptive subtest names.** Use `tc.name` or a descriptive string field so `go test -run TestFoo/CaseA` still works. Where the input itself is self-describing (e.g., a string literal), use `tc.input` as the subtest name.

**t.Parallel() inside subtests.** Existing tests already call `t.Parallel()` at the top level. For subtests, call `t.Parallel()` inside each `t.Run()` after capturing loop variables to avoid closure capture bugs.

**Target files and consolidation plan:**
- `models_test.go`: `TestRegistryResolve_*` (3→1), `TestRegistryLen_*` (2→1), `TestProbeAlias_*` (5→1), `TestDiscover_*` (3→1), `TestNewRegistry_*` (2→1)
- `claude_test.go`: `TestParseBlockingOutput_*` (4→1), `TestSanitizeModelID_*` (2→1)
- `logger_test.go`: 8 functions → 2-3 table-driven tests
- `debug_test.go`: `TestSanitizeHeaders_*` (2→1), `TestDebugMiddleware_*` header tests (3→1)
- `handler_test.go`: `TestHandlerChatCompletions_*` error paths (3→1)
- `config_test.go`: `TestLoad_*` variants (9→2-3)

## Risks / Trade-offs

[Subtests silently skipped] → Use `t.Run()` correctly; run tests after each file change to verify subtest names appear in output.

[Loop variable capture] → Always copy loop variable into a local before `t.Parallel()` (Go 1.22+ handles this automatically, but explicit copies are harmless).
