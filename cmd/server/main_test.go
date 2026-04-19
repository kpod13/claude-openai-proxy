package main

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kpod13/claude-openai-proxy/internal/proxy"
	"github.com/stretchr/testify/require"
)

var (
	errPortInUse = errors.New("port in use")
)

func TestVersionFlag(t *testing.T) {
	var buf bytes.Buffer

	cmd := newRootCmd(&buf)
	cmd.SetArgs([]string{"--version"})

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestHelpFlag(t *testing.T) {
	var buf bytes.Buffer

	cmd := newRootCmd(&buf)
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)
	require.Contains(t, buf.String(), "claude-openai-proxy")
}

func TestCompletionZsh(t *testing.T) {
	var buf bytes.Buffer

	cmd := newRootCmd(&buf)
	cmd.SetArgs([]string{"completion", "zsh"})

	err := cmd.Execute()
	require.NoError(t, err)
	require.Contains(t, buf.String(), "zsh")
}

func TestCompletionBash(t *testing.T) {
	var buf bytes.Buffer

	cmd := newRootCmd(&buf)
	cmd.SetArgs([]string{"completion", "bash"})

	err := cmd.Execute()
	require.NoError(t, err)
	require.True(t, buf.Len() > 0)
}

func TestCompletionFish(t *testing.T) {
	var buf bytes.Buffer

	cmd := newRootCmd(&buf)
	cmd.SetArgs([]string{"completion", "fish"})

	err := cmd.Execute()
	require.NoError(t, err)
	require.True(t, buf.Len() > 0)
}

func TestCompletionPowershell(t *testing.T) {
	var buf bytes.Buffer

	cmd := newRootCmd(&buf)
	cmd.SetArgs([]string{"completion", "powershell"})

	err := cmd.Execute()
	require.NoError(t, err)
	require.True(t, buf.Len() > 0)
}

func TestCompletionUnknownShell(t *testing.T) {
	var buf bytes.Buffer

	cmd := newRootCmd(&buf)
	cmd.SetArgs([]string{"completion", "fish-and-chips"})

	err := cmd.Execute()
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "unsupported shell"))
}

func TestRunE_ConfigMissing(t *testing.T) {
	var buf bytes.Buffer

	cmd := newRootCmd(&buf)
	cmd.SetArgs([]string{"--config", "/nonexistent/path/config.yaml"})

	err := cmd.Execute()
	require.Error(t, err)
}

func TestRunE_NoModels(t *testing.T) {
	// Write a config with empty aliases so Discover returns immediately without
	// spawning any claude subprocesses.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("aliases: []\n"), 0o600))

	var buf bytes.Buffer

	cmd := newRootCmd(&buf)
	cmd.SetArgs([]string{"--config", cfgPath})

	err := cmd.Execute()
	require.ErrorIs(t, err, errNoModels)
}

func TestRunE_ServerStart(t *testing.T) {
	// Use injected deps: a registry with one model and a serve function that
	// returns http.ErrServerClosed immediately (simulates a clean shutdown).
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("aliases: [sonnet]\n"), 0o600))

	reg := proxy.NewRegistry(map[string]string{
		"sonnet":            "claude-sonnet-4-6",
		"claude-sonnet-4-6": "claude-sonnet-4-6",
	})

	deps := serverDeps{
		discover: func(_ []string) *proxy.Registry { return reg },
		serve:    func(_ *http.Server) error { return http.ErrServerClosed },
	}

	var buf bytes.Buffer

	cmd := newRootCmdWith(&buf, deps)
	cmd.SetArgs([]string{"--config", cfgPath})

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestRunE_ServeError(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("aliases: [sonnet]\n"), 0o600))

	reg := proxy.NewRegistry(map[string]string{"sonnet": "claude-sonnet-4-6"})

	deps := serverDeps{
		discover: func(_ []string) *proxy.Registry { return reg },
		serve:    func(_ *http.Server) error { return errPortInUse },
	}

	var buf bytes.Buffer

	cmd := newRootCmdWith(&buf, deps)
	cmd.SetArgs([]string{"--config", cfgPath})

	err := cmd.Execute()
	require.Error(t, err)
}

func TestRunE_WithVerboseAndQuietFlags(t *testing.T) {
	// --quiet takes precedence over --verbose; RunE should still reach errNoModels.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("aliases: []\n"), 0o600))

	var buf bytes.Buffer

	cmd := newRootCmd(&buf)
	cmd.SetArgs([]string{"--verbose", "--quiet", "--log-format", "json", "--config", cfgPath})

	err := cmd.Execute()
	require.ErrorIs(t, err, errNoModels)
}
