package proxy

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// Registry holds discovered model IDs keyed by both full ID and alias.
type Registry struct {
	mu     sync.RWMutex
	models map[string]string // key -> full model ID (key may be alias or full ID)
	list   []ModelObject
}

// cliProbeResult is the minimal JSON shape we need from claude --output-format json.
type cliProbeResult struct {
	ModelUsage map[string]json.RawMessage `json:"modelUsage"`
}

// Discover probes the claude CLI concurrently for each alias and returns a populated Registry.
// Aliases that fail to resolve are silently skipped. Logs a fatal error if no models are found.
func Discover(aliases []string) *Registry {
	type result struct {
		alias   string
		fullID  string
	}

	ch := make(chan result, len(aliases))
	var wg sync.WaitGroup

	for _, alias := range aliases {
		wg.Add(1)
		go func(a string) {
			defer wg.Done()
			fullID, err := probeAlias(a)
			if err != nil {
				// partial failure: skip this alias
				return
			}
			ch <- result{alias: a, fullID: fullID}
		}(alias)
	}

	wg.Wait()
	close(ch)

	reg := &Registry{
		models: make(map[string]string),
	}

	seen := map[string]bool{}
	for r := range ch {
		reg.models[r.alias] = r.fullID
		reg.models[r.fullID] = r.fullID
		if !seen[r.fullID] {
			seen[r.fullID] = true
			reg.list = append(reg.list, ModelObject{
				ID:      r.fullID,
				Object:  "model",
				Created: 1700000000,
				OwnedBy: "anthropic",
			})
		}
	}

	return reg
}

// probeAlias runs a minimal claude invocation to resolve an alias to its full model ID.
func probeAlias(alias string) (string, error) {
	cmd := exec.Command("claude", "--print", "--output-format", "json", "--model", alias,
		"--no-session-persistence", ".")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("probe %s: %w", alias, err)
	}

	var probe cliProbeResult
	// The output may have a leading status line; find the JSON object.
	raw := strings.TrimSpace(string(out))
	start := strings.Index(raw, "{")
	if start == -1 {
		return "", fmt.Errorf("probe %s: no JSON in output", alias)
	}
	if err := json.Unmarshal([]byte(raw[start:]), &probe); err != nil {
		return "", fmt.Errorf("probe %s: parse error: %w", alias, err)
	}

	for fullID := range probe.ModelUsage {
		return fullID, nil
	}
	return "", fmt.Errorf("probe %s: modelUsage is empty", alias)
}

// Resolve returns the full model ID for a given name (full ID or alias).
func (r *Registry) Resolve(name string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if fullID, ok := r.models[name]; ok {
		return fullID, nil
	}
	return "", fmt.Errorf("unknown model %q", name)
}

// List returns all discovered models as ModelObject entries.
func (r *Registry) List() []ModelObject {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ModelObject, len(r.list))
	copy(out, r.list)
	return out
}

// Len returns the number of unique models in the registry.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.list)
}
