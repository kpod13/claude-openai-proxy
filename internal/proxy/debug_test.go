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

func TestDebugMiddleware_Headers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		setReqHdr  func(r *http.Request)
		setRespHdr func(w http.ResponseWriter)
		check      func(t *testing.T, out string)
	}{
		{
			name:      "logs request headers",
			setReqHdr: func(r *http.Request) { r.Header.Set("Content-Type", "application/json") },
			check: func(t *testing.T, out string) {
				t.Helper()
				require.Contains(t, out, "request")
				require.Contains(t, out, "Content-Type")
			},
		},
		{
			name:      "masks Authorization in request",
			setReqHdr: func(r *http.Request) { r.Header.Set("Authorization", "Bearer super-secret-token") },
			check: func(t *testing.T, out string) {
				t.Helper()
				require.Contains(t, out, "Bearer ***")
				require.NotContains(t, out, "super-secret-token")
			},
		},
		{
			name:       "logs response headers",
			setRespHdr: func(w http.ResponseWriter) { w.Header().Set("Content-Type", "application/json") },
			check: func(t *testing.T, out string) {
				t.Helper()
				require.Contains(t, out, "response")
				require.Contains(t, out, "Content-Type")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tc.setRespHdr != nil {
					tc.setRespHdr(w)
				}

				w.WriteHeader(http.StatusOK)
			})
			handler := DebugMiddleware(debugLogger(&buf))(inner)

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/models", http.NoBody)
			if tc.setReqHdr != nil {
				tc.setReqHdr(req)
			}

			handler.ServeHTTP(httptest.NewRecorder(), req)
			tc.check(t, buf.String())
		})
	}
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

// --- maskAuthorization ---

func TestMaskAuthorization(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{"Bearer token123", "Bearer ***"},
		{"Basic abc==", "Basic ***"},
		{"token123", "***"},
		{"", "***"},
		{"Bearer ", "***"},
	}

	for _, tc := range cases {
		require.Equal(t, tc.want, maskAuthorization(tc.input), "input: %q", tc.input)
	}
}

// --- sanitizeHeaders ---

func TestSanitizeHeaders(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		headers http.Header
		check   func(t *testing.T, result headersValue)
	}{
		{
			name: "masks Authorization",
			headers: http.Header{
				"Authorization": {"Bearer secret-token"},
				"Content-Type":  {"application/json"},
			},
			check: func(t *testing.T, result headersValue) {
				t.Helper()
				require.Equal(t, []string{"Bearer ***"}, result["Authorization"])
				require.Equal(t, []string{"application/json"}, result["Content-Type"])
			},
		},
		{
			name: "passthrough non-auth",
			headers: http.Header{
				"X-Custom": {"my-value"},
				"Accept":   {"text/html"},
			},
			check: func(t *testing.T, result headersValue) {
				t.Helper()
				require.Equal(t, []string{"my-value"}, result["X-Custom"])
				require.Equal(t, []string{"text/html"}, result["Accept"])
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.check(t, sanitizeHeaders(tc.headers))
		})
	}
}
