package proxy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

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
