package proxy

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func makeRegistry(entries map[string]string) *Registry {
	reg := &Registry{models: make(map[string]string)}
	seen := map[string]bool{}

	for key, fullID := range entries {
		reg.models[key] = fullID
		if !seen[fullID] {
			seen[fullID] = true
			reg.list = append(reg.list, ModelObject{
				ID:      fullID,
				Object:  "model",
				Created: 0,
				OwnedBy: "anthropic",
			})
		}
	}

	return reg
}

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   map[string]string
		wantLen int
		check   func(t *testing.T, reg *Registry)
	}{
		{
			name: "deduplicates models",
			input: map[string]string{
				"sonnet":            "claude-sonnet-4-6",
				"claude-sonnet-4-6": "claude-sonnet-4-6",
				"haiku":             "claude-haiku-4-5",
				"claude-haiku-4-5":  "claude-haiku-4-5",
			},
			wantLen: 2,
			check: func(t *testing.T, reg *Registry) {
				t.Helper()

				got, err := reg.Resolve("sonnet")
				require.NoError(t, err)
				require.Equal(t, "claude-sonnet-4-6", got)

				got, err = reg.Resolve("claude-haiku-4-5")
				require.NoError(t, err)
				require.Equal(t, "claude-haiku-4-5", got)
			},
		},
		{
			name:    "empty",
			input:   map[string]string{},
			wantLen: 0,
			check: func(t *testing.T, reg *Registry) {
				t.Helper()
				require.Empty(t, reg.List())
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reg := NewRegistry(tc.input)
			require.Equal(t, tc.wantLen, reg.Len())
			tc.check(t, reg)
		})
	}
}

func TestRegistryResolve(t *testing.T) {
	t.Parallel()

	reg := makeRegistry(map[string]string{
		"sonnet":            "claude-sonnet-4-6",
		"claude-sonnet-4-6": "claude-sonnet-4-6",
	})

	cases := []struct {
		name    string
		input   string
		wantID  string
		wantErr bool
	}{
		{"full ID", "claude-sonnet-4-6", "claude-sonnet-4-6", false},
		{"alias", "sonnet", "claude-sonnet-4-6", false},
		{"unknown", "gpt-4", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := reg.Resolve(tc.input)
			if tc.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantID, got)
		})
	}
}

func TestRegistryLen(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		entries map[string]string
		want    int
	}{
		{
			name: "one unique model",
			entries: map[string]string{
				"sonnet":            "claude-sonnet-4-6",
				"claude-sonnet-4-6": "claude-sonnet-4-6",
			},
			want: 1,
		},
		{
			name:    "empty",
			entries: map[string]string{},
			want:    0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reg := makeRegistry(tc.entries)
			require.Equal(t, tc.want, reg.Len())
		})
	}
}

func TestRegistryList(t *testing.T) {
	reg := makeRegistry(map[string]string{
		"sonnet":            "claude-sonnet-4-6",
		"claude-sonnet-4-6": "claude-sonnet-4-6",
		"haiku":             "claude-haiku-4-5",
		"claude-haiku-4-5":  "claude-haiku-4-5",
	})

	list := reg.List()
	require.Len(t, list, 2)

	for _, m := range list {
		require.Equal(t, "anthropic", m.OwnedBy)
		require.Equal(t, "model", m.Object)
	}
}

// --- probeAlias ---

func probeEchoCommand(ctx context.Context, _ string, _ ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "echo", `{"modelUsage":{"claude-sonnet-4-6":{}}}`)
}

func TestProbeAlias(t *testing.T) {
	cases := []struct {
		name    string
		cmd     func(context.Context, string, ...string) *exec.Cmd
		wantID  string
		wantErr bool
	}{
		{
			name:   "success",
			cmd:    probeEchoCommand,
			wantID: "claude-sonnet-4-6",
		},
		{
			name: "command fails",
			cmd: func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.CommandContext(ctx, "false")
			},
			wantErr: true,
		},
		{
			name: "no JSON",
			cmd: func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.CommandContext(ctx, "echo", "no json here")
			},
			wantErr: true,
		},
		{
			name: "invalid JSON",
			cmd: func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.CommandContext(ctx, "echo", `{not valid json}`)
			},
			wantErr: true,
		},
		{
			name: "empty model usage",
			cmd: func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.CommandContext(ctx, "echo", `{"modelUsage":{}}`)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			orig := newCommand
			newCommand = tc.cmd

			t.Cleanup(func() { newCommand = orig })

			got, err := probeAlias("sonnet")
			if tc.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantID, got)
		})
	}
}

// --- Discover ---

func TestDiscover(t *testing.T) {
	cases := []struct {
		name    string
		cmd     func(context.Context, string, ...string) *exec.Cmd
		aliases []string
		wantLen int
		wantID  string
	}{
		{
			name:    "success",
			cmd:     probeEchoCommand,
			aliases: []string{"sonnet"},
			wantLen: 1,
			wantID:  "claude-sonnet-4-6",
		},
		{
			name: "failed alias skipped",
			cmd: func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.CommandContext(ctx, "false")
			},
			aliases: []string{"sonnet", "haiku"},
			wantLen: 0,
		},
		{
			name:    "empty aliases",
			aliases: []string{},
			wantLen: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.cmd != nil {
				orig := newCommand
				newCommand = tc.cmd

				t.Cleanup(func() { newCommand = orig })
			}

			reg := Discover(tc.aliases)
			require.Equal(t, tc.wantLen, reg.Len())

			if tc.wantID != "" {
				got, err := reg.Resolve(tc.wantID)
				require.NoError(t, err)
				require.Equal(t, tc.wantID, got)
			}
		})
	}
}
