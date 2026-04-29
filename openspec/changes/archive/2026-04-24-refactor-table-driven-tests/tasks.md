## 1. models_test.go

- [x] 1.1 Consolidate `TestRegistryResolve_*` (3 functions) into one table-driven `TestRegistryResolve`
- [x] 1.2 Consolidate `TestRegistryLen_*` (2 functions) into one table-driven `TestRegistryLen`
- [x] 1.3 Consolidate `TestProbeAlias_*` (5 functions) into one table-driven `TestProbeAlias`
- [x] 1.4 Consolidate `TestDiscover_*` (3 functions) into one table-driven `TestDiscover`
- [x] 1.5 Consolidate `TestNewRegistry_*` (2 functions) into one table-driven `TestNewRegistry`

## 2. claude_test.go

- [x] 2.1 Consolidate `TestParseBlockingOutput_*` (4 functions) into one table-driven `TestParseBlockingOutput`
- [x] 2.2 Consolidate `TestSanitizeModelID_*` (2 functions) into one table-driven `TestSanitizeModelID`

## 3. logger_test.go

- [x] 3.1 Consolidate level and format tests into table-driven `TestNewLogger` covering all level/format/quiet combinations
- [x] 3.2 Consolidate remaining logger behavior tests using `t.Run()` where applicable

## 4. debug_test.go

- [x] 4.1 Consolidate `TestSanitizeHeaders_MasksAuthorization` and `TestSanitizeHeaders_PassthroughNonAuth` into one table-driven `TestSanitizeHeaders`
- [x] 4.2 Consolidate `TestDebugMiddleware_LogsRequestHeaders`, `TestDebugMiddleware_MasksAuthorizationInRequestLog`, and `TestDebugMiddleware_LogsResponseHeaders` into one table-driven `TestDebugMiddleware_Headers`

## 5. handler_test.go

- [x] 5.1 Consolidate `TestHandlerChatCompletions_*` error path functions into one table-driven test

## 6. config_test.go

- [x] 6.1 Consolidate `TestLoad_*` variants (9 functions) into table-driven `TestLoad` groups

## 7. Verification

- [x] 7.1 Run `go test ./...` and confirm all tests pass
- [x] 7.2 Run `golangci-lint run` and confirm no new lint issues
- [x] 7.3 Confirm coverage remains above 70%
