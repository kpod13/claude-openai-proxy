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

	"github.com/stretchr/testify/require"
	"github.com/kpod13/claude-openai-proxy/internal/ratelimit"
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
func (b *brokenWriter) WriteHeader(code int)         { b.code = code }
func (b *brokenWriter) Write(_ []byte) (int, error) { return 0, errWriteFailed }

// --- serializeMessages ---

func TestSerializeMessages_AllRoles(t *testing.T) {
	msgs := []Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
		{Role: "user", Content: "Bye"},
	}

	got, err := serializeMessages(msgs)
	require.NoError(t, err)

	wantLines := []string{
		"[System]: You are helpful.",
		"[User]: Hello",
		"[Assistant]: Hi there",
		"[User]: Bye",
	}

	for _, line := range wantLines {
		require.Contains(t, got, line)
	}
}

func TestSerializeMessages_UnsupportedRole(t *testing.T) {
	msgs := []Message{
		{Role: "tool", Content: "some tool result"},
	}

	_, err := serializeMessages(msgs)
	require.Error(t, err)
}

// --- Handler.Models ---

func TestHandlerModels(t *testing.T) {
	reg := makeRegistry(map[string]string{
		"sonnet":            "claude-sonnet-4-6",
		"claude-sonnet-4-6": "claude-sonnet-4-6",
	})
	h := &Handler{Registry: reg}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/models", http.NoBody)

	w := httptest.NewRecorder()

	h.Models(w, req)

	resp := w.Result()

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var list ModelList

	err := json.NewDecoder(resp.Body).Decode(&list)
	require.NoError(t, err)

	require.Equal(t, "list", list.Object)
	require.Len(t, list.Data, 1)
	require.Equal(t, "claude-sonnet-4-6", list.Data[0].ID)
	require.Equal(t, "anthropic", list.Data[0].OwnedBy)
}

// --- Handler.ChatCompletions error paths ---

func TestHandlerChatCompletions_MalformedJSON(t *testing.T) {
	reg := makeRegistry(map[string]string{"sonnet": "claude-sonnet-4-6"})
	h := &Handler{Registry: reg}

	body := strings.NewReader("{not json}")
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/chat/completions", body)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	h.ChatCompletions(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandlerChatCompletions_UnknownModel(t *testing.T) {
	reg := makeRegistry(map[string]string{"sonnet": "claude-sonnet-4-6"})
	h := &Handler{Registry: reg}

	body := strings.NewReader(`{"model":"gpt-4","messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/chat/completions", body)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	h.ChatCompletions(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandlerChatCompletions_BadRole(t *testing.T) {
	reg := makeRegistry(map[string]string{"sonnet": "claude-sonnet-4-6"})
	h := &Handler{Registry: reg}

	body := strings.NewReader(`{"model":"sonnet","messages":[{"role":"tool","content":"hi"}]}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/chat/completions", body)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	h.ChatCompletions(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

// --- handleBlocking ---

func TestHandleBlocking_Success(t *testing.T) {
	reg := makeRegistry(map[string]string{"sonnet": "claude-sonnet-4-6"})
	h := &Handler{
		Registry: reg,
		RunBlocking: func(_ context.Context, _, _ string) (*CLIResult, error) {
			return &CLIResult{Text: "Hello!", InputTokens: 5, OutputTokens: 3}, nil
		},
	}

	body := strings.NewReader(`{"model":"sonnet","messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/chat/completions", body)

	w := httptest.NewRecorder()

	h.ChatCompletions(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp ChatResponse

	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	require.Equal(t, "Hello!", resp.Choices[0].Message.Content)
	require.Equal(t, "stop", resp.Choices[0].FinishReason)
	require.Equal(t, 5, resp.Usage.PromptTokens)
	require.Equal(t, 3, resp.Usage.CompletionTokens)
}

func TestHandleBlocking_Error(t *testing.T) {
	reg := makeRegistry(map[string]string{"sonnet": "claude-sonnet-4-6"})
	h := &Handler{
		Registry: reg,
		RunBlocking: func(_ context.Context, _, _ string) (*CLIResult, error) {
			return nil, errFakeClaude
		},
	}

	body := strings.NewReader(`{"model":"sonnet","messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/chat/completions", body)

	w := httptest.NewRecorder()

	h.ChatCompletions(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
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

func TestHandleStreaming_Success(t *testing.T) {
	reg := makeRegistry(map[string]string{"sonnet": "claude-sonnet-4-6"})
	h := &Handler{
		Registry:    reg,
		RunStreaming: makeStreamingChunks("Hello", " world"),
	}

	body := strings.NewReader(`{"model":"sonnet","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/chat/completions", body)

	w := httptest.NewRecorder()

	h.ChatCompletions(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "text/event-stream")

	// Parse SSE lines and collect content deltas.
	var contents []string

	scanner := bufio.NewScanner(w.Body)
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
}

func TestHandleStreaming_RunError(t *testing.T) {
	reg := makeRegistry(map[string]string{"sonnet": "claude-sonnet-4-6"})
	h := &Handler{
		Registry: reg,
		RunStreaming: func(_ context.Context, _, _ string) (<-chan StreamChunk, error) {
			return nil, errFakeStreamFail
		},
	}

	body := strings.NewReader(`{"model":"sonnet","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/chat/completions", body)

	w := httptest.NewRecorder()

	h.ChatCompletions(w, req)

	require.Contains(t, w.Body.String(), "error")
}

func TestModels_EncodeError(_ *testing.T) {
	reg := makeRegistry(map[string]string{"sonnet": "claude-sonnet-4-6"})
	h := &Handler{Registry: reg}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/models", http.NoBody)

	h.Models(newBrokenWriter(), req)
}

func TestHandleBlocking_DefaultRunBlocking(_ *testing.T) {
	// Tests the `if h.RunBlocking == nil { runBlocking = RunBlocking }` branch.
	// The real RunBlocking will fail (no claude CLI), exercising the nil-check path.
	reg := makeRegistry(map[string]string{"sonnet": "claude-sonnet-4-6"})
	h := &Handler{Registry: reg} // RunBlocking is nil → falls back to package-level

	body := strings.NewReader(`{"model":"sonnet","messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/chat/completions", body)
	w := httptest.NewRecorder()

	// Call handleBlocking directly with the resolved model ID.
	h.handleBlocking(w, req, "claude-sonnet-4-6", "[User]: hi\n")
	// claude CLI not available → 500 error; covers the nil-check branch.
}

func TestHandleStreaming_DefaultRunStreaming(_ *testing.T) {
	// Tests the `if h.RunStreaming == nil { runStreaming = RunStreaming }` branch.
	reg := makeRegistry(map[string]string{"sonnet": "claude-sonnet-4-6"})
	h := &Handler{Registry: reg} // RunStreaming is nil → falls back to package-level

	body := strings.NewReader(`{"model":"sonnet","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/chat/completions", body)
	w := httptest.NewRecorder()

	h.handleStreaming(w, req, "claude-sonnet-4-6", "[User]: hi\n")
	// claude CLI not available → error written to stream; covers the nil-check branch.
}

func TestHandleBlocking_EncodeError(_ *testing.T) {
	reg := makeRegistry(map[string]string{"sonnet": "claude-sonnet-4-6"})
	h := &Handler{
		Registry: reg,
		RunBlocking: func(_ context.Context, _, _ string) (*CLIResult, error) {
			return &CLIResult{Text: "ok"}, nil
		},
	}

	body := strings.NewReader(`{"model":"sonnet","messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/chat/completions", body)

	h.handleBlocking(newBrokenWriter(), req, "claude-sonnet-4-6", "[User]: hi\n")
}

func TestHandleStreaming_NoFlusher(t *testing.T) {
	reg := makeRegistry(map[string]string{"sonnet": "claude-sonnet-4-6"})
	h := &Handler{Registry: reg}

	body := strings.NewReader(`{"model":"sonnet","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/chat/completions", body)
	w := newBrokenWriter() // does not implement http.Flusher

	h.handleStreaming(w, req, "claude-sonnet-4-6", "[User]: hi\n")
	require.Equal(t, http.StatusInternalServerError, w.code)
}

func TestHandleStreaming_ChunkError(t *testing.T) {
	reg := makeRegistry(map[string]string{"sonnet": "claude-sonnet-4-6"})
	h := &Handler{
		Registry: reg,
		RunStreaming: func(_ context.Context, _, _ string) (<-chan StreamChunk, error) {
			ch := make(chan StreamChunk, 1)

			ch <- StreamChunk{Err: errFakeMidStream}

			close(ch)

			return ch, nil
		},
	}

	body := strings.NewReader(`{"model":"sonnet","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/chat/completions", body)

	w := httptest.NewRecorder()

	h.ChatCompletions(w, req)

	require.Contains(t, w.Body.String(), "error")
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
