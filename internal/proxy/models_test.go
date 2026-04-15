package proxy

import (
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
