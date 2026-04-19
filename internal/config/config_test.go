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

func TestLoad_InvalidYAML_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "bad.yaml", "listen: [invalid yaml")

	_, err := Load(path)
	require.Error(t, err)
}

func TestLoad_SearchPath_ValidFile_ReturnsCfg(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "search.yaml", `listen: "0.0.0.0:7070"`)

	orig := searchPathsFn
	searchPathsFn = func() []string { return []string{path} }

	t.Cleanup(func() { searchPathsFn = orig })

	cfg, err := Load("")
	require.NoError(t, err)
	require.Equal(t, "0.0.0.0:7070", cfg.Listen)
}

func TestLoad_SearchPath_InvalidYAML_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "broken.yaml", "listen: [bad")

	orig := searchPathsFn
	searchPathsFn = func() []string { return []string{path} }

	t.Cleanup(func() { searchPathsFn = orig })

	_, err := Load("")
	require.Error(t, err)
}

func TestLoad_RateLimit_Parsed(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config.yaml", `
rate_limit:
  requests_per_minute: 60
  tokens_per_minute: 10000
`)

	cfg, err := Load(path)
	require.NoError(t, err)

	require.Equal(t, 60, cfg.RateLimit.RequestsPerMinute)
	require.Equal(t, 10000, cfg.RateLimit.TokensPerMinute)
}

func TestLoad_RateLimit_AbsentUsesDefaults(t *testing.T) {
	cfg, err := Load("")
	require.NoError(t, err)

	require.Equal(t, 0, cfg.RateLimit.RequestsPerMinute)
	require.Equal(t, 0, cfg.RateLimit.TokensPerMinute)
}
