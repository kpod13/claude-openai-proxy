package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// RateLimit holds optional rate limiting configuration.
// Zero values for both fields mean rate limiting is disabled.
type RateLimit struct {
	RequestsPerMinute int `yaml:"requests_per_minute"`
	TokensPerMinute   int `yaml:"tokens_per_minute"`
}

// Config holds server configuration loaded from a YAML file.
type Config struct {
	Listen    string    `yaml:"listen"`
	Aliases   []string  `yaml:"aliases"`
	RateLimit RateLimit `yaml:"rate_limit"`
}

// defaultConfig returns built-in defaults used when no config file is found.
func defaultConfig() Config {
	return Config{
		Listen:  "127.0.0.1:8080",
		Aliases: []string{"opus", "sonnet", "haiku"},
	}
}

// Load loads configuration from the given explicit path, or searches standard
// locations when path is empty. Returns built-in defaults if no file is found.
//
// Search order (first match wins):
//  1. explicit path (error if given but missing)
//  2. ~/.claude-code-openai-server.yaml
//  3. /etc/claude-code-openai-server/config.yaml
//  4. built-in defaults
func Load(path string) (*Config, error) {
	if path != "" {
		return loadFile(path)
	}

	for _, p := range searchPaths() {
		cfg, err := loadFile(p)
		if err == nil {
			return cfg, nil
		}

		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	def := defaultConfig()

	return &def, nil
}

// searchPathsFn is the function used to get search paths; replaced in tests.
var (
	searchPathsFn = defaultSearchPaths
)

// searchPaths delegates to searchPathsFn so tests can inject custom paths.
func searchPaths() []string { return searchPathsFn() }

// defaultSearchPaths returns the standard config file locations in priority order.
func defaultSearchPaths() []string {
	paths := []string{"/etc/claude-code-openai-server/config.yaml"}

	home, err := os.UserHomeDir()
	if err == nil {
		paths = append([]string{filepath.Join(home, ".claude-code-openai-server.yaml")}, paths...)
	}

	return paths
}

// loadFile reads and decodes a YAML config file, merging over built-in defaults.
func loadFile(path string) (*Config, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("config: open %s: %w", path, err)
	}

	defer func() { _ = f.Close() }()

	cfg := defaultConfig()

	err = yaml.NewDecoder(f).Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	return &cfg, nil
}
