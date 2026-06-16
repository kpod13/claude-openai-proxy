package proxy

import (
	"context"
	"os/exec"
	"testing"

	"github.com/kpod13/claude-openai-proxy/internal/config"
	"github.com/stretchr/testify/require"
)

func TestPermissionArgs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		perm config.Permission
		want []string
	}{
		{
			name: "default policy emits nothing",
			perm: config.Permission{Mode: "default"},
			want: nil,
		},
		{
			name: "empty policy emits nothing",
			perm: config.Permission{},
			want: nil,
		},
		{
			name: "mode selected",
			perm: config.Permission{Mode: "acceptEdits"},
			want: []string{"--permission-mode", "acceptEdits"},
		},
		{
			name: "allowed tools",
			perm: config.Permission{Mode: "default", AllowedTools: []string{"Write", "Edit"}},
			want: []string{"--allowedTools", "Write", "Edit"},
		},
		{
			name: "disallowed tools",
			perm: config.Permission{DisallowedTools: []string{"Bash(rm *)"}},
			want: []string{"--disallowedTools", "Bash(rm *)"},
		},
		{
			name: "add dirs",
			perm: config.Permission{AddDirs: []string{"/srv/work", "/tmp/x"}},
			want: []string{"--add-dir", "/srv/work", "--add-dir", "/tmp/x"},
		},
		{
			name: "combined",
			perm: config.Permission{
				Mode:            "acceptEdits",
				AllowedTools:    []string{"Write"},
				DisallowedTools: []string{"Bash(rm *)"},
				AddDirs:         []string{"/srv/work"},
			},
			want: []string{
				"--permission-mode", "acceptEdits",
				"--allowedTools", "Write",
				"--disallowedTools", "Bash(rm *)",
				"--add-dir", "/srv/work",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, PermissionArgs(&tc.perm))
		})
	}
}

// captureCommand records the args passed to newCommand and returns a command
// that emits the given stdout so the runner completes.
func captureCommand(captured *[]string, stdout string) func(context.Context, string, ...string) *exec.Cmd {
	return func(ctx context.Context, _ string, args ...string) *exec.Cmd {
		*captured = append([]string(nil), args...)

		return exec.CommandContext(ctx, "echo", stdout)
	}
}

const (
	blockingStdout = `{"result":"hi","usage":{"input_tokens":1,"output_tokens":1}}`
	imagesStdout   = `{"type":"result","result":"hi","usage":{"input_tokens":1,"output_tokens":1}}`
	streamStdout   = `{"type":"assistant","message":{"content":[{"type":"text","text":"hi"}]}}` + "\n" + `{"type":"result"}`
)

func TestRunnersEmitPermissionFlags(t *testing.T) {
	perm := []string{"--permission-mode", "acceptEdits", "--allowedTools", "Write"}

	cases := []struct {
		name   string
		stdout string
		run    func(perm []string)
	}{
		{
			name:   "blocking",
			stdout: blockingStdout,
			run: func(perm []string) {
				_, _ = BlockingRunner(perm)(context.Background(), "claude-sonnet-4-6", "hi")
			},
		},
		{
			name:   "blocking images",
			stdout: imagesStdout,
			run: func(perm []string) {
				_, _ = BlockingImagesRunner(perm)(context.Background(), "claude-sonnet-4-6", "hi")
			},
		},
		{
			name:   "streaming",
			stdout: streamStdout,
			run: func(perm []string) {
				ch, _ := StreamingRunner(perm)(context.Background(), "claude-sonnet-4-6", "hi")
				for range ch { //nolint:revive // drain channel
				}
			},
		},
		{
			name:   "streaming images",
			stdout: streamStdout,
			run: func(perm []string) {
				ch, _ := StreamingImagesRunner(perm)(context.Background(), "claude-sonnet-4-6", "hi")
				for range ch { //nolint:revive // drain channel
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var captured []string

			orig := newCommand
			newCommand = captureCommand(&captured, tc.stdout)

			t.Cleanup(func() { newCommand = orig })

			tc.run(perm)

			require.GreaterOrEqual(t, len(captured), len(perm))
			require.Equal(t, perm, captured[len(captured)-len(perm):])
		})
	}
}

func TestDefaultRunnersEmitNoPermissionFlags(t *testing.T) {
	cases := []struct {
		name   string
		stdout string
		run    func()
	}{
		{
			name:   "blocking",
			stdout: blockingStdout,
			run: func() {
				_, _ = RunBlocking(context.Background(), "claude-sonnet-4-6", "hi")
			},
		},
		{
			name:   "blocking images",
			stdout: imagesStdout,
			run: func() {
				_, _ = RunBlockingImages(context.Background(), "claude-sonnet-4-6", "hi")
			},
		},
		{
			name:   "streaming",
			stdout: streamStdout,
			run: func() {
				ch, _ := RunStreaming(context.Background(), "claude-sonnet-4-6", "hi")
				for range ch { //nolint:revive // drain channel
				}
			},
		},
		{
			name:   "streaming images",
			stdout: streamStdout,
			run: func() {
				ch, _ := RunStreamingImages(context.Background(), "claude-sonnet-4-6", "hi")
				for range ch { //nolint:revive // drain channel
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var captured []string

			orig := newCommand
			newCommand = captureCommand(&captured, tc.stdout)

			t.Cleanup(func() { newCommand = orig })

			tc.run()

			require.NotContains(t, captured, "--permission-mode")
			require.NotContains(t, captured, "--allowedTools")
			require.NotContains(t, captured, "--disallowedTools")
			require.NotContains(t, captured, "--add-dir")
		})
	}
}
