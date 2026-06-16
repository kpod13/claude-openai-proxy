package proxy

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	// probeTimeout bounds a single alias probe so a hung claude CLI cannot wedge
	// startup indefinitely.
	probeTimeout = 30 * time.Second
)

// Sentinel errors for model discovery.
var (
	errProbeNoJSON    = errors.New("no JSON in probe output")
	errProbeNoModels  = errors.New("modelUsage is empty in probe output")
	errUnknownModel   = errors.New("unknown model")
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
// Aliases that fail to resolve are silently skipped.
func Discover(aliases []string) *Registry {
	type result struct {
		alias  string
		fullID string
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

	models := make(map[string]string)
	for r := range ch {
		// Register both the alias and the full ID so either resolves.
		models[r.alias] = r.fullID
		models[r.fullID] = r.fullID
	}

	return NewRegistry(models)
}

// probeAlias runs a minimal claude invocation to resolve an alias to its full model ID.
func probeAlias(alias string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	cmd := newCommand(ctx,
		"claude", "--print", "--output-format", "json", "--model", alias,
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
		return "", fmt.Errorf("probe %s: %w", alias, errProbeNoJSON)
	}

	err = json.Unmarshal([]byte(raw[start:]), &probe)
	if err != nil {
		return "", fmt.Errorf("probe %s: parse error: %w", alias, err)
	}

	for fullID := range probe.ModelUsage {
		return fullID, nil
	}

	return "", fmt.Errorf("probe %s: %w", alias, errProbeNoModels)
}

// Resolve returns the full model ID for a given name (full ID or alias).
func (r *Registry) Resolve(name string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if fullID, ok := r.models[name]; ok {
		return fullID, nil
	}

	return "", fmt.Errorf("%w: %q", errUnknownModel, name)
}

// List returns all discovered models as ModelObject entries.
func (r *Registry) List() []ModelObject {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]ModelObject, len(r.list))
	copy(out, r.list)

	return out
}

// NewRegistry creates a Registry from a map of alias → full model ID.
func NewRegistry(aliasToFullID map[string]string) *Registry {
	reg := &Registry{models: make(map[string]string)}
	seen := make(map[string]bool)

	for alias, fullID := range aliasToFullID {
		reg.models[alias] = fullID

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

	// Map iteration order is random; sort so /v1/models output is stable across restarts.
	slices.SortFunc(reg.list, func(a, b ModelObject) int {
		return cmp.Compare(a.ID, b.ID)
	})

	return reg
}

// Len returns the number of unique models in the registry.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.list)
}
