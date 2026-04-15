package proxy

import (
	"testing"
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
		"sonnet":           "claude-sonnet-4-6",
		"claude-sonnet-4-6": "claude-sonnet-4-6",
	})

	got, err := reg.Resolve("claude-sonnet-4-6")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "claude-sonnet-4-6" {
		t.Errorf("got %q, want %q", got, "claude-sonnet-4-6")
	}
}

func TestRegistryResolve_Alias(t *testing.T) {
	reg := makeRegistry(map[string]string{
		"sonnet":           "claude-sonnet-4-6",
		"claude-sonnet-4-6": "claude-sonnet-4-6",
	})

	got, err := reg.Resolve("sonnet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "claude-sonnet-4-6" {
		t.Errorf("got %q, want %q", got, "claude-sonnet-4-6")
	}
}

func TestRegistryResolve_Unknown(t *testing.T) {
	reg := makeRegistry(map[string]string{
		"sonnet": "claude-sonnet-4-6",
	})

	_, err := reg.Resolve("gpt-4")
	if err == nil {
		t.Fatal("expected error for unknown model, got nil")
	}
}

func TestRegistryList(t *testing.T) {
	reg := makeRegistry(map[string]string{
		"sonnet":           "claude-sonnet-4-6",
		"claude-sonnet-4-6": "claude-sonnet-4-6",
		"haiku":            "claude-haiku-4-5",
		"claude-haiku-4-5": "claude-haiku-4-5",
	})

	list := reg.List()
	if len(list) != 2 {
		t.Errorf("expected 2 unique models, got %d", len(list))
	}
	for _, m := range list {
		if m.OwnedBy != "anthropic" {
			t.Errorf("expected owned_by=anthropic, got %q", m.OwnedBy)
		}
		if m.Object != "model" {
			t.Errorf("expected object=model, got %q", m.Object)
		}
	}
}
