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
				Created: 1700000000,
				OwnedBy: "anthropic",
			})
		}
	}

	return reg
}

func TestRegistryResolve_FullID(t *testing.T) {
	reg := makeRegistry(map[string]string{
		"sonnet":            "claude-sonnet-4-6",
		"claude-sonnet-4-6": "claude-sonnet-4-6",
	})

	got, err := reg.Resolve("claude-sonnet-4-6")
	require.NoError(t, err)

	require.Equal(t, "claude-sonnet-4-6", got)
}

func TestRegistryResolve_Alias(t *testing.T) {
	reg := makeRegistry(map[string]string{
		"sonnet":            "claude-sonnet-4-6",
		"claude-sonnet-4-6": "claude-sonnet-4-6",
	})

	got, err := reg.Resolve("sonnet")
	require.NoError(t, err)

	require.Equal(t, "claude-sonnet-4-6", got)
}

func TestRegistryResolve_Unknown(t *testing.T) {
	reg := makeRegistry(map[string]string{
		"sonnet": "claude-sonnet-4-6",
	})

	_, err := reg.Resolve("gpt-4")
	require.Error(t, err)
}

func TestRegistryLen(t *testing.T) {
	reg := makeRegistry(map[string]string{
		"sonnet":            "claude-sonnet-4-6",
		"claude-sonnet-4-6": "claude-sonnet-4-6",
	})

	require.Equal(t, 1, reg.Len())
}

func TestRegistryLen_Empty(t *testing.T) {
	reg := makeRegistry(map[string]string{})
	require.Equal(t, 0, reg.Len())
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

func TestProbeAlias_Success(t *testing.T) {
	orig := newCommand
	newCommand = probeEchoCommand

	t.Cleanup(func() { newCommand = orig })

	got, err := probeAlias("sonnet")
	require.NoError(t, err)
	require.Equal(t, "claude-sonnet-4-6", got)
}

func TestProbeAlias_CommandFails(t *testing.T) {
	orig := newCommand
	newCommand = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "false")
	}

	t.Cleanup(func() { newCommand = orig })

	_, err := probeAlias("sonnet")
	require.Error(t, err)
}

func TestProbeAlias_NoJSON(t *testing.T) {
	orig := newCommand
	newCommand = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "echo", "no json here")
	}

	t.Cleanup(func() { newCommand = orig })

	_, err := probeAlias("sonnet")
	require.Error(t, err)
}

func TestProbeAlias_InvalidJSON(t *testing.T) {
	orig := newCommand
	newCommand = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "echo", `{not valid json}`)
	}

	t.Cleanup(func() { newCommand = orig })

	_, err := probeAlias("sonnet")
	require.Error(t, err)
}

func TestProbeAlias_EmptyModelUsage(t *testing.T) {
	orig := newCommand
	newCommand = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "echo", `{"modelUsage":{}}`)
	}

	t.Cleanup(func() { newCommand = orig })

	_, err := probeAlias("sonnet")
	require.Error(t, err)
}

// --- Discover ---

func TestDiscover_Success(t *testing.T) {
	orig := newCommand
	newCommand = probeEchoCommand

	t.Cleanup(func() { newCommand = orig })

	reg := Discover([]string{"sonnet"})
	require.Equal(t, 1, reg.Len())

	got, err := reg.Resolve("claude-sonnet-4-6")
	require.NoError(t, err)
	require.Equal(t, "claude-sonnet-4-6", got)
}

func TestDiscover_FailedAliasSkipped(t *testing.T) {
	orig := newCommand
	newCommand = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "false")
	}

	t.Cleanup(func() { newCommand = orig })

	reg := Discover([]string{"sonnet", "haiku"})
	require.Equal(t, 0, reg.Len())
}

func TestDiscover_Empty(t *testing.T) {
	reg := Discover([]string{})
	require.Equal(t, 0, reg.Len())
}
