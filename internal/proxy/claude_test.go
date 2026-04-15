package proxy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseBlockingOutput_Valid(t *testing.T) {
	raw := []byte(`{"type":"result","result":"Hello!","usage":{"input_tokens":10,"output_tokens":5}}`)

	got, err := parseBlockingOutput(raw)
	require.NoError(t, err)

	require.Equal(t, "Hello!", got.Text)
	require.Equal(t, 10, got.InputTokens)
	require.Equal(t, 5, got.OutputTokens)
}

func TestParseBlockingOutput_WithLeadingText(t *testing.T) {
	// The CLI sometimes emits a status line before the JSON object.
	raw := []byte("- Loading...\n{\"type\":\"result\",\"result\":\"Hi\",\"usage\":{\"input_tokens\":1,\"output_tokens\":2}}")

	got, err := parseBlockingOutput(raw)
	require.NoError(t, err)

	require.Equal(t, "Hi", got.Text)
}

func TestParseBlockingOutput_NoJSON(t *testing.T) {
	raw := []byte("something went wrong")

	_, err := parseBlockingOutput(raw)
	require.Error(t, err)
}

func TestParseBlockingOutput_InvalidJSON(t *testing.T) {
	raw := []byte("{not valid json}")

	_, err := parseBlockingOutput(raw)
	require.Error(t, err)
}
