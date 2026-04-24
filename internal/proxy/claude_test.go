package proxy

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseBlockingOutput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    []byte
		wantText string
		wantIn   int
		wantOut  int
		wantErr  bool
	}{
		{
			name:     "valid",
			input:    []byte(`{"type":"result","result":"Hello!","usage":{"input_tokens":10,"output_tokens":5}}`),
			wantText: "Hello!",
			wantIn:   10,
			wantOut:  5,
		},
		{
			name:     "with leading text",
			input:    []byte("- Loading...\n{\"type\":\"result\",\"result\":\"Hi\",\"usage\":{\"input_tokens\":1,\"output_tokens\":2}}"),
			wantText: "Hi",
		},
		{
			name:    "no JSON",
			input:   []byte("something went wrong"),
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   []byte("{not valid json}"),
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseBlockingOutput(tc.input)
			if tc.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantText, got.Text)

			if tc.wantIn > 0 {
				require.Equal(t, tc.wantIn, got.InputTokens)
				require.Equal(t, tc.wantOut, got.OutputTokens)
			}
		})
	}
}

func TestSanitizeModelID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input   string
		wantErr bool
	}{
		{"claude-sonnet-4-6", false},
		{"claude-haiku-4-5", false},
		{"claude-opus-4-6", false},
		{"sonnet", false},
		{"claude sonnet", true},
		{"claude/sonnet", true},
		{"../../etc/passwd", true},
		{"model;rm -rf", true},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			got, err := sanitizeModelID(tc.input)
			if tc.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.input, got)
		})
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
