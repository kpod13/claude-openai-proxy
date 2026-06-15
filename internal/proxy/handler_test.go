package proxy

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kpod13/claude-openai-proxy/internal/ratelimit"
	"github.com/stretchr/testify/require"
)

var (
	errFakeClaude     = errors.New("claude failed")
	errFakeStreamFail = errors.New("stream failed")
	errFakeMidStream  = errors.New("mid-stream error")
	errWriteFailed    = errors.New("write failed")
)

// brokenWriter is a ResponseWriter whose Write always fails.
type brokenWriter struct {
	header http.Header
	code   int
}

func newBrokenWriter() *brokenWriter {
	return &brokenWriter{header: make(http.Header)}
}

func (b *brokenWriter) Header() http.Header         { return b.header }
func (b *brokenWriter) WriteHeader(code int)        { b.code = code }
func (b *brokenWriter) Write(_ []byte) (int, error) { return 0, errWriteFailed }

// --- serializeMessages ---

func TestSerializeMessages(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		msgs     []Message
		wantErr  bool
		wantText []string
	}{
		{
			name: "all roles",
			msgs: []Message{
				{Role: "system", Content: "You are helpful."},
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there"},
				{Role: "user", Content: "Bye"},
			},
			wantText: []string{
				"[System]: You are helpful.",
				"[User]: Hello",
				"[Assistant]: Hi there",
				"[User]: Bye",
			},
		},
		{
			name:    "unsupported role",
			msgs:    []Message{{Role: "tool", Content: "some tool result"}},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := serializeMessages(tc.msgs)
			if tc.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)

			for _, line := range tc.wantText {
				require.Contains(t, got, line)
			}
		})
	}
}

// --- Handler.Models ---

func TestHandlerModels(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		writer http.ResponseWriter
		check  func(t *testing.T, w http.ResponseWriter)
	}{
		{
			name:   "success",
			writer: httptest.NewRecorder(),
			check: func(t *testing.T, w http.ResponseWriter) {
				t.Helper()

				rec, ok := w.(*httptest.ResponseRecorder)
				require.True(t, ok)

				resp := rec.Result()

				defer func() { _ = resp.Body.Close() }()

				require.Equal(t, http.StatusOK, resp.StatusCode)

				var list ModelList

				err := json.NewDecoder(resp.Body).Decode(&list)
				require.NoError(t, err)

				require.Equal(t, "list", list.Object)
				require.Len(t, list.Data, 1)
				require.Equal(t, "claude-sonnet-4-6", list.Data[0].ID)
				require.Equal(t, "anthropic", list.Data[0].OwnedBy)
			},
		},
		{
			name:   "encode error tolerated",
			writer: newBrokenWriter(),
			check:  func(_ *testing.T, _ http.ResponseWriter) {},
		},
	}

	reg := NewRegistry(map[string]string{
		"sonnet":            "claude-sonnet-4-6",
		"claude-sonnet-4-6": "claude-sonnet-4-6",
	})

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := &Handler{Registry: reg}
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/models", http.NoBody)

			h.Models(tc.writer, req)
			tc.check(t, tc.writer)
		})
	}
}

// --- Handler.ChatCompletions error paths ---

func TestHandlerChatCompletions_BadRequest(t *testing.T) {
	t.Parallel()

	reg := NewRegistry(map[string]string{"sonnet": "claude-sonnet-4-6"})
	h := &Handler{Registry: reg}

	cases := []struct {
		name string
		body string
	}{
		{"malformed JSON", "{not json}"},
		{"unknown model", `{"model":"gpt-4","messages":[{"role":"user","content":"hi"}]}`},
		{"bad role", `{"model":"sonnet","messages":[{"role":"tool","content":"hi"}]}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/chat/completions", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			h.ChatCompletions(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

// --- handleBlocking ---

func TestHandleBlocking(t *testing.T) {
	t.Parallel()

	body := `{"model":"sonnet","messages":[{"role":"user","content":"hi"}]}`

	cases := []struct {
		name        string
		runBlocking func(context.Context, string, string) (*CLIResult, error)
		writer      http.ResponseWriter
		direct      bool // call h.handleBlocking directly (skips dispatcher)
		check       func(t *testing.T, w http.ResponseWriter)
	}{
		{
			name: "success",
			runBlocking: func(_ context.Context, _, _ string) (*CLIResult, error) {
				return &CLIResult{Text: "Hello!", InputTokens: 5, OutputTokens: 3}, nil
			},
			writer: httptest.NewRecorder(),
			check: func(t *testing.T, w http.ResponseWriter) {
				t.Helper()

				rec, ok := w.(*httptest.ResponseRecorder)
				require.True(t, ok)
				require.Equal(t, http.StatusOK, rec.Code)

				var resp ChatResponse

				err := json.NewDecoder(rec.Body).Decode(&resp)
				require.NoError(t, err)

				require.Equal(t, "Hello!", resp.Choices[0].Message.Content)
				require.Equal(t, "stop", resp.Choices[0].FinishReason)
				require.Equal(t, 5, resp.Usage.PromptTokens)
				require.Equal(t, 3, resp.Usage.CompletionTokens)
			},
		},
		{
			name: "run error",
			runBlocking: func(_ context.Context, _, _ string) (*CLIResult, error) {
				return nil, errFakeClaude
			},
			writer: httptest.NewRecorder(),
			check: func(t *testing.T, w http.ResponseWriter) {
				t.Helper()

				rec, ok := w.(*httptest.ResponseRecorder)
				require.True(t, ok)
				require.Equal(t, http.StatusInternalServerError, rec.Code)
			},
		},
		{
			// Tests fallback to package-level RunBlocking when h.RunBlocking is nil.
			// claude CLI is not available → covers the nil-check branch.
			name:    "default RunBlocking nil-check",
			direct:  true,
			writer:  httptest.NewRecorder(),
			check:   func(_ *testing.T, _ http.ResponseWriter) {},
		},
		{
			name: "encode error",
			runBlocking: func(_ context.Context, _, _ string) (*CLIResult, error) {
				return &CLIResult{Text: "ok"}, nil
			},
			direct: true,
			writer: newBrokenWriter(),
			check:  func(_ *testing.T, _ http.ResponseWriter) {},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reg := NewRegistry(map[string]string{"sonnet": "claude-sonnet-4-6"})
			h := &Handler{Registry: reg, RunBlocking: tc.runBlocking}

			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/chat/completions", strings.NewReader(body))

			if tc.direct {
				h.handleBlocking(tc.writer, req, "claude-sonnet-4-6", "[User]: hi\n")
			} else {
				h.ChatCompletions(tc.writer, req)
			}

			tc.check(t, tc.writer)
		})
	}
}

// --- handleStreaming ---

func makeStreamingChunks(chunks ...string) func(context.Context, string, string) (<-chan StreamChunk, error) {
	return func(_ context.Context, _, _ string) (<-chan StreamChunk, error) {
		ch := make(chan StreamChunk, len(chunks))

		for _, c := range chunks {
			ch <- StreamChunk{Text: c}
		}

		close(ch)

		return ch, nil
	}
}

func TestHandleStreaming(t *testing.T) {
	t.Parallel()

	body := `{"model":"sonnet","stream":true,"messages":[{"role":"user","content":"hi"}]}`

	cases := []struct {
		name         string
		runStreaming func(context.Context, string, string) (<-chan StreamChunk, error)
		writer       http.ResponseWriter
		direct       bool
		check        func(t *testing.T, w http.ResponseWriter)
	}{
		{
			name:         "success",
			runStreaming: makeStreamingChunks("Hello", " world"),
			writer:       httptest.NewRecorder(),
			check: func(t *testing.T, w http.ResponseWriter) {
				t.Helper()

				rec, ok := w.(*httptest.ResponseRecorder)
				require.True(t, ok)
				require.Equal(t, http.StatusOK, rec.Code)
				require.Contains(t, rec.Header().Get("Content-Type"), "text/event-stream")

				var contents []string

				scanner := bufio.NewScanner(rec.Body)
				for scanner.Scan() {
					line := scanner.Text()
					if !strings.HasPrefix(line, "data: ") {
						continue
					}

					data := strings.TrimPrefix(line, "data: ")
					if data == "[DONE]" {
						break
					}

					var chunk ChatCompletionChunk

					err := json.Unmarshal([]byte(data), &chunk)
					require.NoError(t, err)

					for _, c := range chunk.Choices {
						if c.Delta.Content != "" {
							contents = append(contents, c.Delta.Content)
						}
					}
				}

				require.Equal(t, []string{"Hello", " world"}, contents)
			},
		},
		{
			name: "run error",
			runStreaming: func(_ context.Context, _, _ string) (<-chan StreamChunk, error) {
				return nil, errFakeStreamFail
			},
			writer: httptest.NewRecorder(),
			check: func(t *testing.T, w http.ResponseWriter) {
				t.Helper()

				rec, ok := w.(*httptest.ResponseRecorder)
				require.True(t, ok)
				require.Contains(t, rec.Body.String(), "error")
			},
		},
		{
			name: "chunk error",
			runStreaming: func(_ context.Context, _, _ string) (<-chan StreamChunk, error) {
				ch := make(chan StreamChunk, 1)

				ch <- StreamChunk{Err: errFakeMidStream}

				close(ch)

				return ch, nil
			},
			writer: httptest.NewRecorder(),
			check: func(t *testing.T, w http.ResponseWriter) {
				t.Helper()

				rec, ok := w.(*httptest.ResponseRecorder)
				require.True(t, ok)
				require.Contains(t, rec.Body.String(), "error")
			},
		},
		{
			// Fallback to package-level RunStreaming. claude CLI not available → error written.
			name:   "default RunStreaming nil-check",
			direct: true,
			writer: httptest.NewRecorder(),
			check:  func(_ *testing.T, _ http.ResponseWriter) {},
		},
		{
			name:   "no flusher",
			direct: true,
			writer: newBrokenWriter(),
			check: func(t *testing.T, w http.ResponseWriter) {
				t.Helper()

				bw, ok := w.(*brokenWriter)
				require.True(t, ok)
				require.Equal(t, http.StatusInternalServerError, bw.code)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reg := NewRegistry(map[string]string{"sonnet": "claude-sonnet-4-6"})
			h := &Handler{Registry: reg, RunStreaming: tc.runStreaming}

			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/chat/completions", strings.NewReader(body))

			if tc.direct {
				h.handleStreaming(tc.writer, req, "claude-sonnet-4-6", "[User]: hi\n")
			} else {
				h.ChatCompletions(tc.writer, req)
			}

			tc.check(t, tc.writer)
		})
	}
}

// --- Rate limit integration ---

// okHandler is a trivial inner handler that always returns 200.
var (
	okHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
)

func sendRateLimitReq(t *testing.T, handler http.Handler, bearerKey string) *httptest.ResponseRecorder {
	t.Helper()

	body := `{"model":"sonnet","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost, "/v1/chat/completions",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	if bearerKey != "" {
		req.Header.Set("Authorization", "Bearer "+bearerKey)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	return w
}

func TestRateLimitMiddleware_RPMExceeded_Returns429WithHeaders(t *testing.T) {
	handler := ratelimit.Middleware(ratelimit.New(2, 0))(okHandler)

	// First two requests: allowed, headers present.
	for i := range 2 {
		w := sendRateLimitReq(t, handler, "test-key")
		require.Equal(t, http.StatusOK, w.Code, "request %d should be allowed", i+1)
		require.Equal(t, "2", w.Header().Get("X-Ratelimit-Limit-Requests"))
		require.NotEmpty(t, w.Header().Get("X-Ratelimit-Remaining-Requests"))
		require.NotEmpty(t, w.Header().Get("X-Ratelimit-Reset-Requests"))
	}

	// Third request exceeds RPM=2.
	w := sendRateLimitReq(t, handler, "test-key")
	require.Equal(t, http.StatusTooManyRequests, w.Code)
	require.NotEmpty(t, w.Header().Get("Retry-After"))
	require.Equal(t, "2", w.Header().Get("X-Ratelimit-Limit-Requests"))
	require.Equal(t, "0", w.Header().Get("X-Ratelimit-Remaining-Requests"))

	var errBody struct {
		Error struct {
			Type string `json:"type"`
			Code string `json:"code"`
		} `json:"error"`
	}

	err := json.NewDecoder(w.Body).Decode(&errBody)
	require.NoError(t, err)
	require.Equal(t, "requests", errBody.Error.Type)
	require.Equal(t, "rate_limit_exceeded", errBody.Error.Code)
}
