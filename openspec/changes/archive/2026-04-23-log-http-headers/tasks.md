## 1. Implementation

- [x] 1.1 Add `maskAuthorization(value string) string` to `debug.go` — returns `<scheme> ***` for Authorization values, passes other values through as-is
- [x] 1.2 Add `sanitizeHeaders(headers http.Header) map[string][]string` to `debug.go` — copies headers applying masking to `Authorization`
- [x] 1.3 In `DebugMiddleware`, log request headers (after the `"request"` log entry) using `sanitizeHeaders(r.Header)`
- [x] 1.4 In `DebugMiddleware`, log response headers (after `next.ServeHTTP`) using `sanitizeHeaders(rec.Header())`

## 2. Tests

- [x] 2.1 Test `maskAuthorization`: verify `Bearer token123` → `Bearer ***`, `Basic abc` → `Basic ***`, value without scheme → `***`, empty string → `***`
- [x] 2.2 Test `sanitizeHeaders`: verify `Authorization` is masked, other headers pass through unchanged
- [x] 2.3 Test `DebugMiddleware`: request with headers — verify `request-headers` appears in log
- [x] 2.4 Test `DebugMiddleware`: verify `Authorization` is masked in log (`Bearer ***`, not the real token)
- [x] 2.5 Test `DebugMiddleware`: verify response headers are logged (`response-headers`)
