package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/kpod13/claude-openai-proxy/internal/proxy"
	"github.com/stretchr/testify/require"
)

var (
	errPortInUse = errors.New("port in use")
)

func TestRootCmd_Flags(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		args         []string
		wantContains string
	}{
		{name: "version", args: []string{"--version"}},
		{name: "help", args: []string{"--help"}, wantContains: "claude-openai-proxy"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			cmd := newRootCmd(&buf)
			cmd.SetOut(&buf)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			require.NoError(t, err)

			if tc.wantContains != "" {
				require.Contains(t, buf.String(), tc.wantContains)
			}
		})
	}
}

func TestCompletion(t *testing.T) {
	t.Parallel()

	cases := []struct {
		shell        string
		wantErr      bool
		wantContains string
	}{
		{shell: "zsh", wantContains: "zsh"},
		{shell: "bash"},
		{shell: "fish"},
		{shell: "powershell"},
		{shell: "fish-and-chips", wantErr: true, wantContains: "unsupported shell"},
	}

	for _, tc := range cases {
		t.Run(tc.shell, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			cmd := newRootCmd(&buf)
			cmd.SetArgs([]string{"completion", tc.shell})

			err := cmd.Execute()

			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantContains)

				return
			}

			require.NoError(t, err)

			if tc.wantContains != "" {
				require.Contains(t, buf.String(), tc.wantContains)
			} else {
				require.Positive(t, buf.Len())
			}
		})
	}
}

// writeConfig writes a config.yaml in a temp dir and returns its path.
func writeConfig(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

// stubRegistry returns a registry resolving "sonnet" → "claude-sonnet-4-6".
func stubRegistry() *proxy.Registry {
	return proxy.NewRegistry(map[string]string{
		"sonnet":            "claude-sonnet-4-6",
		"claude-sonnet-4-6": "claude-sonnet-4-6",
	})
}

func TestRunE(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		configYAML string // empty → use missing path instead
		missingCfg bool
		extraArgs  []string
		deps       *serverDeps
		wantErrIs  error
		wantErr    bool
	}{
		{
			name:       "config missing",
			missingCfg: true,
			wantErr:    true,
		},
		{
			name:       "no models",
			configYAML: "aliases: []\n",
			wantErrIs:  errNoModels,
		},
		{
			name:       "server starts",
			configYAML: "aliases: [sonnet]\n",
			deps: &serverDeps{
				discover: func(_ []string) *proxy.Registry { return stubRegistry() },
				serve:    func(_ *http.Server) error { return http.ErrServerClosed },
			},
		},
		{
			name:       "verbose flag",
			configYAML: "aliases: [sonnet]\n",
			extraArgs:  []string{"--verbose"},
			deps: &serverDeps{
				discover: func(_ []string) *proxy.Registry { return stubRegistry() },
				serve:    func(_ *http.Server) error { return http.ErrServerClosed },
			},
		},
		{
			name:       "serve error",
			configYAML: "aliases: [sonnet]\n",
			deps: &serverDeps{
				discover: func(_ []string) *proxy.Registry { return stubRegistry() },
				serve:    func(_ *http.Server) error { return errPortInUse },
			},
			wantErr: true,
		},
		{
			// --quiet takes precedence over --verbose; RunE still reaches errNoModels.
			name:       "verbose and quiet flags",
			configYAML: "aliases: []\n",
			extraArgs:  []string{"--verbose", "--quiet", "--log-format", "json"},
			wantErrIs:  errNoModels,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var cfgPath string
			if tc.missingCfg {
				cfgPath = "/nonexistent/path/config.yaml"
			} else {
				cfgPath = writeConfig(t, tc.configYAML)
			}

			var buf bytes.Buffer

			var cmd = newRootCmd(&buf)
			if tc.deps != nil {
				cmd = newRootCmdWith(&buf, *tc.deps)
			}

			args := append([]string{}, tc.extraArgs...)
			args = append(args, "--config", cfgPath)
			cmd.SetArgs(args)

			err := cmd.Execute()

			switch {
			case tc.wantErrIs != nil:
				require.ErrorIs(t, err, tc.wantErrIs)
			case tc.wantErr:
				require.Error(t, err)
			default:
				require.NoError(t, err)
			}
		})
	}
}

// TestRunE_LogsModelNames is kept separate because it captures os.Stderr,
// which is process-global and cannot be exercised under t.Parallel().
func TestRunE_LogsModelNames(t *testing.T) {
	cfgPath := writeConfig(t, "aliases: [sonnet]\n")

	deps := serverDeps{
		discover: func(_ []string) *proxy.Registry { return stubRegistry() },
		serve:    func(_ *http.Server) error { return http.ErrServerClosed },
	}

	r, w, err := os.Pipe()
	require.NoError(t, err)

	oldStderr := os.Stderr
	os.Stderr = w

	var cmdBuf bytes.Buffer

	cmd := newRootCmdWith(&cmdBuf, deps)
	cmd.SetArgs([]string{"--config", cfgPath})

	execErr := cmd.Execute()

	os.Stderr = oldStderr

	require.NoError(t, w.Close())

	var logBuf bytes.Buffer

	_, _ = io.Copy(&logBuf, r)

	require.NoError(t, r.Close())
	require.NoError(t, execErr)
	require.Contains(t, logBuf.String(), "claude-sonnet-4-6")
}
