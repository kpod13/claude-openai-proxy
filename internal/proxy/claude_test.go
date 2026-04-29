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

func TestVersion(t *testing.T) {
	cases := []struct {
		name    string
		cmd     func(context.Context, string, ...string) *exec.Cmd
		want    string
		wantErr bool
	}{
		{
			name: "success",
			cmd: func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.CommandContext(ctx, "echo", "1.2.3 (Claude Code)")
			},
			want: "1.2.3 (Claude Code)",
		},
		{
			name:    "command fails",
			cmd:     failCommand,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			orig := newCommand
			newCommand = tc.cmd

			t.Cleanup(func() { newCommand = orig })

			ver, err := Version(context.Background())

			if tc.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.want, ver)
		})
	}
}

// --- RunBlocking ---

func echoCommand(ctx context.Context, _ string, _ ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "echo", `{"result":"Hello!","usage":{"input_tokens":5,"output_tokens":3}}`)
}

func failCommand(ctx context.Context, _ string, _ ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "false")
}

func TestRunBlocking(t *testing.T) {
	cases := []struct {
		name    string
		cmd     func(context.Context, string, ...string) *exec.Cmd
		model   string
		wantErr bool
		check   func(t *testing.T, got *CLIResult)
	}{
		{
			name:  "success",
			cmd:   echoCommand,
			model: "claude-sonnet-4-6",
			check: func(t *testing.T, got *CLIResult) {
				t.Helper()
				require.Equal(t, "Hello!", got.Text)
				require.Equal(t, 5, got.InputTokens)
			},
		},
		{
			name:    "invalid model",
			model:   "bad model!",
			wantErr: true,
		},
		{
			name:    "command fails",
			cmd:     failCommand,
			model:   "claude-sonnet-4-6",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.cmd != nil {
				orig := newCommand
				newCommand = tc.cmd

				t.Cleanup(func() { newCommand = orig })
			}

			got, err := RunBlocking(context.Background(), tc.model, "hello")

			if tc.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)

			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

// --- RunStreaming ---

func streamEchoCommand(ctx context.Context, _ string, _ ...string) *exec.Cmd {
	line := `{"type":"assistant","message":{"content":[{"type":"text","text":"Hello"}]}}` + "\n" + `{"type":"result"}`

	return exec.CommandContext(ctx, "echo", line)
}

func TestRunStreaming(t *testing.T) {
	cases := []struct {
		name      string
		cmd       func(context.Context, string, ...string) *exec.Cmd
		model     string
		wantErr   bool
		wantTexts []string
	}{
		{
			name:      "success",
			cmd:       streamEchoCommand,
			model:     "claude-sonnet-4-6",
			wantTexts: []string{"Hello"},
		},
		{
			name:    "invalid model",
			model:   "bad model!",
			wantErr: true,
		},
		{
			name: "skips empty and invalid lines",
			cmd: func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
				lines := "\n" +
					"not json at all\n" +
					`{"type":"assistant","message":{"content":[{"type":"text","text":"Hi"}]}}` + "\n" +
					`{"type":"result"}`

				return exec.CommandContext(ctx, "echo", lines)
			},
			model:     "claude-sonnet-4-6",
			wantTexts: []string{"Hi"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.cmd != nil {
				orig := newCommand
				newCommand = tc.cmd

				t.Cleanup(func() { newCommand = orig })
			}

			ch, err := RunStreaming(context.Background(), tc.model, "hello")

			if tc.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)

			var texts []string

			for chunk := range ch {
				require.NoError(t, chunk.Err)
				texts = append(texts, chunk.Text)
			}

			require.Equal(t, tc.wantTexts, texts)
		})
	}
}
