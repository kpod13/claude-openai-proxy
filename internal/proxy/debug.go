package proxy

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// statusRecorder wraps http.ResponseWriter to capture the written status code.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Flush passes through to the underlying ResponseWriter if it supports http.Flusher.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// maskAuthorization replaces the credential part of an Authorization header value with ***.
// The scheme (e.g. "Bearer", "Basic") is preserved. Returns "***" for empty or scheme-only values.
func maskAuthorization(value string) string {
	parts := strings.SplitN(value, " ", 2)
	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
		return "***"
	}

	return parts[0] + " ***"
}

// headersValue is a sanitized header map that implements slog.LogValuer.
// In text format it renders as key=value pairs; in JSON as a nested object.
type headersValue map[string][]string

func (h headersValue) LogValue() slog.Value {
	attrs := make([]slog.Attr, 0, len(h))

	for k, vs := range h {
		attrs = append(attrs, slog.String(k, strings.Join(vs, ", ")))
	}

	return slog.GroupValue(attrs...)
}

// sanitizeHeaders copies headers into a headersValue, masking the Authorization value.
func sanitizeHeaders(headers http.Header) headersValue {
	out := make(headersValue, len(headers))

	for k, vs := range headers {
		if http.CanonicalHeaderKey(k) == "Authorization" {
			masked := make([]string, len(vs))
			for i, v := range vs {
				masked[i] = maskAuthorization(v)
			}

			out[k] = masked
		} else {
			out[k] = vs
		}
	}

	return out
}

// DebugMiddleware returns an HTTP middleware that logs each request and response at DEBUG level.
func DebugMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			log.Debug("request", slog.String("method", r.Method), slog.String("path", r.URL.Path), slog.Any("headers", sanitizeHeaders(r.Header)))

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			log.Debug("response", slog.Int("status", rec.status), slog.Any("duration", time.Since(start)), slog.Any("headers", sanitizeHeaders(rec.Header())))
		})
	}
}

// DebugRunBlocking wraps a RunBlocking function with DEBUG logging.
func DebugRunBlocking(log *slog.Logger, fn func(context.Context, string, string) (*CLIResult, error)) func(context.Context, string, string) (*CLIResult, error) {
	return func(ctx context.Context, model, prompt string) (*CLIResult, error) {
		start := time.Now()

		log.Debug("CLI invoke", "model", model, "prompt_len", len(prompt))

		result, err := fn(ctx, model, prompt)
		if err != nil {
			log.Debug("CLI error", "model", model, "err", err, "duration", time.Since(start))

			return nil, err
		}

		log.Debug("CLI done", "model", model, "input_tokens", result.InputTokens, "output_tokens", result.OutputTokens, "duration", time.Since(start))

		return result, nil
	}
}

// DebugRunStreaming wraps a RunStreaming function with DEBUG logging.
func DebugRunStreaming(log *slog.Logger, fn func(context.Context, string, string) (<-chan StreamChunk, error)) func(context.Context, string, string) (<-chan StreamChunk, error) {
	return func(ctx context.Context, model, prompt string) (<-chan StreamChunk, error) {
		start := time.Now()

		log.Debug("CLI stream", "model", model, "prompt_len", len(prompt))

		ch, err := fn(ctx, model, prompt)
		if err != nil {
			log.Debug("CLI stream error", "model", model, "err", err, "duration", time.Since(start))

			return nil, err
		}

		log.Debug("CLI stream started", "model", model, "duration", time.Since(start))

		return ch, nil
	}
}
