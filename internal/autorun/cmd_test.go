package autorun

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	errLaunchctlFailed = errors.New("launchctl failed")
	errUnloadFailed    = errors.New("unload failed")
	errClaudeNotFound  = errors.New("claude not found")
)

// stubBackend is a Backend that records calls and returns configurable errors.
type stubBackend struct {
	installErr   error
	uninstallErr error
	installed    bool
	uninstalled  bool
}

func (s *stubBackend) Install(_ context.Context, _ InstallConfig) error {
	s.installed = true

	return s.installErr
}

func (s *stubBackend) Uninstall(_ context.Context) error {
	s.uninstalled = true

	return s.uninstallErr
}

func stubVersion(_ context.Context) (string, error) {
	return "1.2.3 (Claude Code)", nil
}

func failVersion(_ context.Context) (string, error) {
	return "", errClaudeNotFound
}

func makeCmd(t *testing.T, stub *stubBackend, getVer func(context.Context) (string, error)) (out *bytes.Buffer, run func(args ...string) error) {
	t.Helper()

	var buf bytes.Buffer

	cmd := newCmdWith(&buf, func() (Backend, error) { return stub, nil }, getVer)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	return &buf, func(args ...string) error {
		cmd.SetArgs(args)

		return cmd.Execute()
	}
}

// --- install subcommand ---

func TestInstallCmd(t *testing.T) {
	cases := []struct {
		name        string
		newBackend  func() (Backend, error)
		getVer      func(context.Context) (string, error)
		preSetup    func(t *testing.T)
		wantErr     bool
		errIs       error
		errContains string
		wantOut     []string
	}{
		{
			name:       "prints version and confirmation",
			newBackend: func() (Backend, error) { return &stubBackend{}, nil },
			getVer:     stubVersion,
			wantOut:    []string{"Claude CLI version: 1.2.3", "Autorun installed for"},
		},
		{
			name:       "version fail prints warning",
			newBackend: func() (Backend, error) { return &stubBackend{}, nil },
			getVer:     failVersion,
			wantOut:    []string{"Warning: could not determine Claude CLI version", "Autorun installed for"},
		},
		{
			name:        "backend error",
			newBackend:  func() (Backend, error) { return &stubBackend{installErr: errLaunchctlFailed}, nil },
			getVer:      stubVersion,
			wantErr:     true,
			errContains: "launchctl failed",
		},
		{
			name:       "unsupported OS",
			newBackend: func() (Backend, error) { return nil, ErrUnsupportedOS },
			getVer:     stubVersion,
			wantErr:    true,
			errIs:      ErrUnsupportedOS,
		},
		{
			name:       "writes default config",
			newBackend: func() (Backend, error) { return &stubBackend{}, nil },
			getVer:     stubVersion,
			wantOut:    []string{"Default config written to"},
		},
		{
			name:       "skips existing config",
			newBackend: func() (Backend, error) { return &stubBackend{}, nil },
			getVer:     stubVersion,
			preSetup: func(t *testing.T) {
				t.Helper()

				_, err := WriteDefaultConfigIfAbsent()
				require.NoError(t, err)
			},
			wantOut: []string{"was not modified"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("HOME", dir)

			if tc.preSetup != nil {
				tc.preSetup(t)
			}

			var buf bytes.Buffer

			cmd := newCmdWith(&buf, tc.newBackend, tc.getVer)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs([]string{"install"})

			err := cmd.Execute()

			if tc.wantErr {
				require.Error(t, err)

				if tc.errIs != nil {
					require.ErrorIs(t, err, tc.errIs)
				}

				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}

				return
			}

			require.NoError(t, err)

			for _, s := range tc.wantOut {
				require.Contains(t, buf.String(), s)
			}
		})
	}
}

// --- uninstall subcommand ---

func TestUninstallCmd(t *testing.T) {
	cases := []struct {
		name        string
		stub        *stubBackend
		wantErr     bool
		errContains string
		wantOut     string
	}{
		{
			name:    "prints confirmation",
			stub:    &stubBackend{},
			wantOut: "Autorun uninstalled",
		},
		{
			name:        "backend error",
			stub:        &stubBackend{uninstallErr: errUnloadFailed},
			wantErr:     true,
			errContains: "unload failed",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf, run := makeCmd(t, tc.stub, stubVersion)

			err := run("uninstall")

			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)

				return
			}

			require.NoError(t, err)

			if tc.wantOut != "" {
				require.Contains(t, buf.String(), tc.wantOut)
			}
		})
	}
}

func TestNewCmd_ReturnsCobraCommand(t *testing.T) {
	cmd := NewCmd(os.Stdout)

	require.NotNil(t, cmd)
	require.Equal(t, "autorun", cmd.Use)
}
