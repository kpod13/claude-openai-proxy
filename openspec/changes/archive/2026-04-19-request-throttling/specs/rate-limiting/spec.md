## ADDED Requirements

### Requirement: Rate limit response headers
The server SHALL include the following headers in every HTTP response from `/v1/chat/completions`:

- `x-ratelimit-limit-requests`: integer, configured RPM limit (omitted when limit is 0/unlimited).
- `x-ratelimit-limit-tokens`: integer, configured TPM limit (omitted when limit is 0/unlimited).
- `x-ratelimit-remaining-requests`: integer, requests remaining in the current 1-minute window.
- `x-ratelimit-remaining-tokens`: integer, tokens remaining in the current 1-minute window.
- `x-ratelimit-reset-requests`: Go duration string (e.g. `"30s"`, `"1m0s"`), time until the requests window resets.
- `x-ratelimit-reset-tokens`: Go duration string, time until the tokens window resets.

Headers SHALL be omitted for any limit dimension that is not configured (value 0).

#### Scenario: Headers present when limits configured
- **WHEN** `rate_limit.requests_per_minute: 60` and `rate_limit.tokens_per_minute: 10000` are set
- **THEN** every `/v1/chat/completions` response includes all six `x-ratelimit-*` headers

#### Scenario: Headers omitted when limits not configured
- **WHEN** `rate_limit` block is absent from the config file
- **THEN** no `x-ratelimit-*` headers are included in responses

#### Scenario: Remaining counts decrease with usage
- **WHEN** a request is processed that consumed 100 prompt tokens
- **THEN** `x-ratelimit-remaining-tokens` is decremented by at least 100

### Requirement: HTTP 429 on rate limit exceeded
When a request would exceed a configured limit the server SHALL:
1. Reject the request immediately (before calling Claude).
2. Return HTTP status `429 Too Many Requests`.
3. Return a `Retry-After` header containing the integer number of seconds until the window resets (minimum 1).
4. Return a JSON body matching the OpenAI error envelope:

```json
{
  "error": {
    "message": "<human-readable description including limit type, limit value, used value>",
    "type": "requests",
    "param": null,
    "code": "rate_limit_exceeded"
  }
}
```

The `type` field SHALL be `"requests"` when the RPM limit is exceeded and `"tokens"` when the TPM limit is exceeded.

#### Scenario: RPM limit exceeded
- **WHEN** a key has made 60 requests in the current minute and `requests_per_minute` is 60
- **THEN** the server returns 429 with `Retry-After` and `"type": "requests"` in the error body

#### Scenario: TPM limit exceeded
- **WHEN** a key has consumed 10000 tokens in the current minute and `tokens_per_minute` is 10000
- **THEN** the server returns 429 with `Retry-After` and `"type": "tokens"` in the error body

#### Scenario: Retry-After value is at least 1
- **WHEN** the window resets in less than 1 second
- **THEN** `Retry-After` is still `1`

### Requirement: Per-API-key rate limiting
Rate limit counters SHALL be scoped to the bearer token value from the `Authorization: Bearer <token>` header. Requests without an `Authorization` header SHALL share a single anonymous bucket.

#### Scenario: Different keys have independent counters
- **WHEN** key-A has exhausted its RPM limit
- **THEN** requests from key-B are still accepted (assuming key-B has remaining quota)

#### Scenario: Unauthenticated requests share one bucket
- **WHEN** multiple requests arrive without an Authorization header
- **THEN** they all decrement the same anonymous bucket

### Requirement: Fixed 1-minute window
Rate limit windows SHALL align to UTC clock minutes (truncated to the minute). Counters reset at each new minute boundary.

#### Scenario: Counter resets at minute boundary
- **WHEN** the current UTC minute increments
- **THEN** `x-ratelimit-remaining-requests` returns to the configured limit for the next request

### Requirement: Rate limiting disabled by default
When the `rate_limit` block is absent from the config file, or when `requests_per_minute` and `tokens_per_minute` are both 0, the server SHALL apply no rate limiting and SHALL NOT return `x-ratelimit-*` headers.

#### Scenario: No config — no limiting
- **WHEN** the server starts without a `rate_limit` config block
- **THEN** all requests are processed without checking limits and no `x-ratelimit-*` headers appear
