## MODIFIED Requirements

### Requirement: Config file format
The config file SHALL use YAML format with the following optional keys:

- `listen` (string): TCP address to bind, e.g. `"127.0.0.1:8080"`.
- `aliases` (array of strings): model aliases to probe at startup, e.g. `["opus", "sonnet", "haiku"]`.
- `rate_limit` (object): optional rate limiting settings. When absent or when both sub-keys are 0, rate limiting is disabled.
  - `requests_per_minute` (integer, default 0): maximum number of requests per key per minute. 0 means unlimited.
  - `tokens_per_minute` (integer, default 0): maximum number of tokens (prompt tokens) per key per minute. 0 means unlimited.

Unrecognised keys SHALL be ignored.

#### Scenario: Valid config parsed
- **WHEN** the config file contains `listen = "0.0.0.0:9090"` and `aliases = ["sonnet"]`
- **THEN** the server binds to `0.0.0.0:9090` and probes only the `sonnet` alias

#### Scenario: Partial config merges with defaults
- **WHEN** the config file contains only `listen = "0.0.0.0:9090"`
- **THEN** the server uses `0.0.0.0:9090` for the address and the default alias list

#### Scenario: Rate limit config parsed
- **WHEN** the config file contains `rate_limit: {requests_per_minute: 60, tokens_per_minute: 10000}`
- **THEN** the server enforces those limits on incoming requests

#### Scenario: Rate limit config absent
- **WHEN** the config file contains no `rate_limit` block
- **THEN** the server starts with rate limiting disabled (no limits enforced)
