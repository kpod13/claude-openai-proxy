package autorun

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultConfigName = ".claude-code-openai-server.yaml"

	// defaultConfigContent is the YAML written when no config file exists yet.
	defaultConfigContent = `# claude-openai-proxy configuration
listen: "127.0.0.1:8080"
aliases:
  - opus
  - sonnet
  - haiku
`
)

// WriteDefaultConfigIfAbsent writes a default YAML config to
// ~/.claude-code-openai-server.yaml only if that file does not already exist.
// It returns (true, nil) when the file was created and (false, nil) when the
// file already existed and was left unchanged.
func WriteDefaultConfigIfAbsent() (created bool, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Errorf("autorun: get home dir: %w", err)
	}

	path := filepath.Join(home, defaultConfigName)

	_, err = os.Stat(path)
	if err == nil {
		return false, nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("autorun: stat config: %w", err)
	}

	err = os.WriteFile(path, []byte(defaultConfigContent), 0o600)
	if err != nil {
		return false, fmt.Errorf("autorun: write default config: %w", err)
	}

	return true, nil
}
