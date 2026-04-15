package proxy

import (
	"testing"
)

func TestParseBlockingOutput_Valid(t *testing.T) {
	raw := []byte(`{"type":"result","result":"Hello!","usage":{"input_tokens":10,"output_tokens":5}}`)

	got, err := parseBlockingOutput(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Text != "Hello!" {
		t.Errorf("Text: got %q, want %q", got.Text, "Hello!")
	}
	if got.InputTokens != 10 {
		t.Errorf("InputTokens: got %d, want 10", got.InputTokens)
	}
	if got.OutputTokens != 5 {
		t.Errorf("OutputTokens: got %d, want 5", got.OutputTokens)
	}
}

func TestParseBlockingOutput_WithLeadingText(t *testing.T) {
	// The CLI sometimes emits a status line before the JSON object.
	raw := []byte("- Loading...\n{\"type\":\"result\",\"result\":\"Hi\",\"usage\":{\"input_tokens\":1,\"output_tokens\":2}}")

	got, err := parseBlockingOutput(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Text != "Hi" {
		t.Errorf("Text: got %q, want %q", got.Text, "Hi")
	}
}

func TestParseBlockingOutput_NoJSON(t *testing.T) {
	raw := []byte("something went wrong")

	_, err := parseBlockingOutput(raw)
	if err == nil {
		t.Fatal("expected error for missing JSON, got nil")
	}
}

func TestParseBlockingOutput_InvalidJSON(t *testing.T) {
	raw := []byte("{not valid json}")

	_, err := parseBlockingOutput(raw)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
