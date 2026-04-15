package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)

	err := os.WriteFile(path, []byte(content), 0o600)
	require.NoError(t, err)

	return path
}

func TestLoad_NoFile_UsesDefaults(t *testing.T) {
	cfg, err := Load("")
	require.NoError(t, err)

	require.Equal(t, "127.0.0.1:8080", cfg.Listen)
	require.Equal(t, []string{"opus", "sonnet", "haiku"}, cfg.Aliases)
}

func TestLoad_ValidYAML_OverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config.yaml", `
listen: "0.0.0.0:9090"
aliases:
  - sonnet
`)

	cfg, err := Load(path)
	require.NoError(t, err)

	require.Equal(t, "0.0.0.0:9090", cfg.Listen)
	require.Equal(t, []string{"sonnet"}, cfg.Aliases)
}

func TestLoad_PartialYAML_MergesWithDefaults(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config.yaml", `listen: "0.0.0.0:9090"`)

	cfg, err := Load(path)
	require.NoError(t, err)

	require.Equal(t, "0.0.0.0:9090", cfg.Listen)
	require.Equal(t, []string{"opus", "sonnet", "haiku"}, cfg.Aliases)
}

func TestLoad_ExplicitPathMissing_ReturnsError(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	require.Error(t, err)
}
