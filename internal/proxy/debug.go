package proxy

import (
	"context"
	"log/slog"
	"net/http"
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

// DebugMiddleware returns an HTTP middleware that logs each request and response at DEBUG level.
func DebugMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			log.Debug("request", "method", r.Method, "path", r.URL.Path)

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			log.Debug("response", "status", rec.status, "duration", time.Since(start))
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
