## 1. Rate Limiter Package

- [x] 1.1 Create `internal/ratelimit/` package with `Limiter` struct and fixed-window counters (RPM + TPM per key)
- [x] 1.2 Implement `Allow(key string, estimatedTokens int) (Info, bool)` method on `Limiter`
- [x] 1.3 Implement `Info` struct carrying limit, remaining, and reset-duration values for requests and tokens
- [x] 1.4 Write unit tests for `Limiter`: RPM enforcement, TPM enforcement, per-key isolation, window reset

## 2. HTTP Middleware

- [x] 2.1 Implement `Middleware(l *Limiter) func(http.Handler) http.Handler` in `internal/ratelimit/`
- [x] 2.2 Middleware sets `x-ratelimit-*` headers on every allowed response
- [x] 2.3 Middleware returns 429 JSON error + `Retry-After` header when `Allow` returns false
- [x] 2.4 Write unit tests for middleware: headers present, 429 body format, Retry-After value

## 3. Config Extension

- [x] 3.1 Add `RateLimit` struct with `RequestsPerMinute` and `TokensPerMinute` int fields to `internal/config/config.go`
- [x] 3.2 Wire `RateLimit` into the `Config` struct with YAML tag `rate_limit`
- [x] 3.3 Update config tests to cover the new `rate_limit` fields (present and absent)

## 4. Wiring

- [x] 4.1 In `cmd/server/main.go`, construct a `*ratelimit.Limiter` from config after load
- [x] 4.2 Wrap the `ChatCompletions` handler with `ratelimit.Middleware` when limits are configured
- [x] 4.3 Leave `/v1/models` and `/healthz` unwrapped

## 5. Integration Smoke Test

- [x] 5.1 Add an integration test (or extend handler_test.go) that sends requests over the RPM limit and asserts 429 + correct headers
