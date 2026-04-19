package proxy

import (
	"context"
	"os/exec"
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

// --- sanitizeModelID ---

func TestSanitizeModelID_Valid(t *testing.T) {
	cases := []string{
		"claude-sonnet-4-6",
		"claude-haiku-4-5",
		"claude-opus-4-6",
		"sonnet",
	}

	for _, tc := range cases {
		got, err := sanitizeModelID(tc)
		require.NoError(t, err)
		require.Equal(t, tc, got)
	}
}

func TestSanitizeModelID_Invalid(t *testing.T) {
	cases := []string{
		"claude sonnet",
		"claude/sonnet",
		"../../etc/passwd",
		"model;rm -rf",
	}

	for _, tc := range cases {
		_, err := sanitizeModelID(tc)
		require.Error(t, err)
	}
}

// --- Version ---

func TestVersion_Success(t *testing.T) {
	orig := newCommand
	newCommand = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "echo", "1.2.3 (Claude Code)")
	}

	t.Cleanup(func() { newCommand = orig })

	ver, err := Version(context.Background())

	require.NoError(t, err)
	require.Equal(t, "1.2.3 (Claude Code)", ver)
}

func TestVersion_CommandFails(t *testing.T) {
	orig := newCommand
	newCommand = failCommand

	t.Cleanup(func() { newCommand = orig })

	_, err := Version(context.Background())

	require.Error(t, err)
}

// --- RunBlocking ---

func echoCommand(ctx context.Context, _ string, _ ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "echo", `{"result":"Hello!","usage":{"input_tokens":5,"output_tokens":3}}`)
}

func failCommand(ctx context.Context, _ string, _ ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "false")
}

func TestRunBlocking_Success(t *testing.T) {
	orig := newCommand
	newCommand = echoCommand

	t.Cleanup(func() { newCommand = orig })

	got, err := RunBlocking(context.Background(), "claude-sonnet-4-6", "hello")
	require.NoError(t, err)
	require.Equal(t, "Hello!", got.Text)
	require.Equal(t, 5, got.InputTokens)
}

func TestRunBlocking_InvalidModel(t *testing.T) {
	_, err := RunBlocking(context.Background(), "bad model!", "hello")
	require.Error(t, err)
}

func TestRunBlocking_CommandFails(t *testing.T) {
	orig := newCommand
	newCommand = failCommand

	t.Cleanup(func() { newCommand = orig })

	_, err := RunBlocking(context.Background(), "claude-sonnet-4-6", "hello")
	require.Error(t, err)
}

// --- RunStreaming ---

func streamEchoCommand(ctx context.Context, _ string, _ ...string) *exec.Cmd {
	line := `{"type":"assistant","message":{"content":[{"type":"text","text":"Hello"}]}}` + "\n" + `{"type":"result"}`

	return exec.CommandContext(ctx, "echo", line)
}

func TestRunStreaming_Success(t *testing.T) {
	orig := newCommand
	newCommand = streamEchoCommand

	t.Cleanup(func() { newCommand = orig })

	ch, err := RunStreaming(context.Background(), "claude-sonnet-4-6", "hello")
	require.NoError(t, err)

	var texts []string

	for chunk := range ch {
		require.NoError(t, chunk.Err)
		texts = append(texts, chunk.Text)
	}

	require.Equal(t, []string{"Hello"}, texts)
}

func TestRunStreaming_InvalidModel(t *testing.T) {
	_, err := RunStreaming(context.Background(), "bad model!", "hello")
	require.Error(t, err)
}

func TestRunStreaming_SkipsEmptyAndInvalidLines(t *testing.T) {
	orig := newCommand
	newCommand = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		// Mix: empty line, invalid JSON, valid assistant message, result.
		lines := "\n" +
			"not json at all\n" +
			`{"type":"assistant","message":{"content":[{"type":"text","text":"Hi"}]}}` + "\n" +
			`{"type":"result"}`

		return exec.CommandContext(ctx, "echo", lines)
	}

	t.Cleanup(func() { newCommand = orig })

	ch, err := RunStreaming(context.Background(), "claude-sonnet-4-6", "hello")
	require.NoError(t, err)

	var texts []string

	for chunk := range ch {
		require.NoError(t, chunk.Err)
		texts = append(texts, chunk.Text)
	}

	require.Equal(t, []string{"Hi"}, texts)
}
