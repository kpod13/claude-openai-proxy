package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantLines := []string{
		"[System]: You are helpful.",
		"[User]: Hello",
		"[Assistant]: Hi there",
		"[User]: Bye",
	}
	for _, line := range wantLines {
		if !strings.Contains(got, line) {
			t.Errorf("expected output to contain %q, got:\n%s", line, got)
		}
	}
}

func TestSerializeMessages_UnsupportedRole(t *testing.T) {
	msgs := []Message{
		{Role: "tool", Content: "some tool result"},
	}

	_, err := serializeMessages(msgs)
	if err == nil {
		t.Fatal("expected error for unsupported role, got nil")
	}
}

// --- Handler.Models ---

func TestHandlerModels(t *testing.T) {
	reg := makeRegistry(map[string]string{
		"sonnet":            "claude-sonnet-4-6",
		"claude-sonnet-4-6": "claude-sonnet-4-6",
	})
	h := &Handler{Registry: reg}

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()
	h.Models(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}

	var list ModelList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if list.Object != "list" {
		t.Errorf("object: got %q, want %q", list.Object, "list")
	}
	if len(list.Data) != 1 {
		t.Errorf("data length: got %d, want 1", len(list.Data))
	}
	if list.Data[0].ID != "claude-sonnet-4-6" {
		t.Errorf("model id: got %q, want %q", list.Data[0].ID, "claude-sonnet-4-6")
	}
	if list.Data[0].OwnedBy != "anthropic" {
		t.Errorf("owned_by: got %q, want %q", list.Data[0].OwnedBy, "anthropic")
	}
}

// --- Handler.ChatCompletions error paths ---

func TestHandlerChatCompletions_MalformedJSON(t *testing.T) {
	reg := makeRegistry(map[string]string{"sonnet": "claude-sonnet-4-6"})
	h := &Handler{Registry: reg}

	body := strings.NewReader("{not json}")
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ChatCompletions(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestHandlerChatCompletions_UnknownModel(t *testing.T) {
	reg := makeRegistry(map[string]string{"sonnet": "claude-sonnet-4-6"})
	h := &Handler{Registry: reg}

	body := strings.NewReader(`{"model":"gpt-4","messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ChatCompletions(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}
