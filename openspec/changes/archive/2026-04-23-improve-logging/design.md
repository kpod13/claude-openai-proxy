## Context

The proxy currently constructs a `*slog.Logger` in `main.go` and passes it only to the server startup sequence. The `proxy.Handler` and the package-level `RunBlocking`/`RunStreaming` functions have no logger access, so even though `--verbose` correctly sets the log level to DEBUG, there are zero DEBUG-level calls anywhere in the request path. Operators see nothing useful when they enable verbose mode.

## Goals / Non-Goals

**Goals:**
- Print discovered model names at startup alongside the count.
- Log each HTTP request (method, path, model) and response (status, latency) at DEBUG level.
- Log each Claude CLI invocation (model, prompt length) and result (token counts, latency) at DEBUG level.
- Ensure `--verbose` exposes this information without any additional configuration.

**Non-Goals:**
- Logging request/response bodies (may contain sensitive user data).
- Structured access logs at INFO level (this is debug observability, not production audit logging).
- Changing the signatures of the exported `RunBlocking` / `RunStreaming` functions.

## Decisions

### 1. Logger field on `proxy.Handler`

Add `Logger *slog.Logger` to `proxy.Handler`. Handlers call `h.Logger.Debug(...)` directly.

**Alternatives considered:**
- *Context value*: Idiomatic but adds boilerplate at every call site; the logger doesn't change per-request.
- *Package-level logger*: Simple but untestable and not concurrent-safe with multiple server instances.
- *Middleware-only logging*: Handles HTTP in/out but gives no visibility into the CLI layer.

The struct field is the standard Go service pattern and keeps the logger close to where it's used.

### 2. CLI logging via Handler wrapper methods

Rather than adding a logger parameter to `RunBlocking` / `RunStreaming` (which would change their public API), `Handler` adds private wrapper methods `runBlocking` / `runStreaming` that call the package-level functions and emit DEBUG logs before and after. The `RunBlocking` / `RunStreaming` func fields on `Handler` remain as test-injection hooks.

**Alternative considered:** Pass `*slog.Logger` as a first argument to `RunBlocking` / `RunStreaming`. Rejected because it changes the exported API and the caller (main.go) would need updating.

### 3. Model names at startup

`proxy.Registry.List()` already returns `[]ModelObject`. `main.go` collects IDs from that slice into a `[]string` and passes it as a log attribute. No new method is needed on `Registry`.

### 4. No-op logger fallback

If `Handler.Logger` is nil (e.g., in existing tests that construct `Handler{}` directly), all debug calls are guarded with `if h.Logger != nil`. This avoids nil-pointer panics without requiring test changes.

## Risks / Trade-offs

- [Verbose output volume] At DEBUG level, every request emits two log lines; busy deployments may produce a lot of output → Mitigation: debug logging is opt-in via `--verbose`; INFO mode is unchanged.
- [Prompt length logging] Logging prompt character length (not content) is safe, but callers should be aware the metric is visible in logs → Mitigation: only length is logged, not content.

## Migration Plan

No migration needed. The change is purely additive:
- `Handler.Logger` defaults to nil (safe no-op).
- No config file changes.
- Existing tests continue to work; new tests cover the debug log paths.
