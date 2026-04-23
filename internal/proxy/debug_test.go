package proxy

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	errFakeDebugCLI    = errors.New("CLI failed")
	errFakeDebugStream = errors.New("stream failed")
)

func debugLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// --- DebugMiddleware ---

func TestDebugMiddleware_LogsRequestAndResponse(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := DebugMiddleware(debugLogger(&buf))(inner)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/models", http.NoBody)

	handler.ServeHTTP(httptest.NewRecorder(), req)

	out := buf.String()

	require.Contains(t, out, "request")
	require.Contains(t, out, "/v1/models")
	require.Contains(t, out, "response")
}

func TestDebugMiddleware_CapturesNon200Status(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	handler := DebugMiddleware(debugLogger(&buf))(inner)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/missing", http.NoBody)

	handler.ServeHTTP(httptest.NewRecorder(), req)

	require.Contains(t, buf.String(), "404")
}

func TestDebugMiddleware_PassesThroughFlusher(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	flushed := false

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		f, ok := w.(http.Flusher)

		require.True(t, ok)

		f.Flush()

		flushed = true
	})

	handler := DebugMiddleware(debugLogger(&buf))(inner)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)

	handler.ServeHTTP(httptest.NewRecorder(), req)

	require.True(t, flushed)
}

// --- DebugRunBlocking ---

func TestDebugRunBlocking_LogsInvokeAndDone(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	wrapped := DebugRunBlocking(debugLogger(&buf), func(_ context.Context, _, _ string) (*CLIResult, error) {
		return &CLIResult{Text: "ok", InputTokens: 10, OutputTokens: 5}, nil
	})

	result, err := wrapped(context.Background(), "claude-sonnet-4-6", "hello")

	require.NoError(t, err)
	require.Equal(t, "ok", result.Text)

	out := buf.String()

	require.Contains(t, out, "CLI invoke")
	require.Contains(t, out, "CLI done")
	require.Contains(t, out, "claude-sonnet-4-6")
}

func TestDebugRunBlocking_LogsError(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	wrapped := DebugRunBlocking(debugLogger(&buf), func(_ context.Context, _, _ string) (*CLIResult, error) {
		return nil, errFakeDebugCLI
	})

	_, err := wrapped(context.Background(), "claude-sonnet-4-6", "hello")

	require.ErrorIs(t, err, errFakeDebugCLI)
	require.Contains(t, buf.String(), "CLI error")
}

// --- DebugRunStreaming ---

func TestDebugRunStreaming_LogsInvokeAndStarted(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	ch := make(chan StreamChunk)

	close(ch)

	wrapped := DebugRunStreaming(debugLogger(&buf), func(_ context.Context, _, _ string) (<-chan StreamChunk, error) {
		return ch, nil
	})

	got, err := wrapped(context.Background(), "claude-sonnet-4-6", "hello")

	require.NoError(t, err)
	require.NotNil(t, got)

	out := buf.String()

	require.Contains(t, out, "CLI stream")
	require.Contains(t, out, "CLI stream started")
}

func TestDebugRunStreaming_LogsError(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	wrapped := DebugRunStreaming(debugLogger(&buf), func(_ context.Context, _, _ string) (<-chan StreamChunk, error) {
		return nil, errFakeDebugStream
	})

	_, err := wrapped(context.Background(), "claude-sonnet-4-6", "hello")

	require.ErrorIs(t, err, errFakeDebugStream)
	require.Contains(t, buf.String(), "CLI stream error")
}
