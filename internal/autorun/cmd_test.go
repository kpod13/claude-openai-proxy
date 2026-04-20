package autorun

import (
	"bytes"
	"context"
	"errors"
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

func TestInstallCmd_PrintsVersionAndConfirmation(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	stub := &stubBackend{}
	buf, run := makeCmd(t, stub, stubVersion)

	err := run("install")

	require.NoError(t, err)
	require.True(t, stub.installed)
	require.Contains(t, buf.String(), "Claude CLI version: 1.2.3")
	require.Contains(t, buf.String(), "Autorun installed for")
}

func TestInstallCmd_VersionFailPrintsWarning(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	stub := &stubBackend{}
	buf, run := makeCmd(t, stub, failVersion)

	err := run("install")

	require.NoError(t, err)
	require.Contains(t, buf.String(), "Warning: could not determine Claude CLI version")
	require.Contains(t, buf.String(), "Autorun installed for")
}

func TestInstallCmd_BackendError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	stub := &stubBackend{installErr: errLaunchctlFailed}
	_, run := makeCmd(t, stub, stubVersion)

	err := run("install")

	require.Error(t, err)
	require.Contains(t, err.Error(), "launchctl failed")
}

func TestInstallCmd_UnsupportedOS(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	var buf bytes.Buffer

	cmd := newCmdWith(&buf, func() (Backend, error) { return nil, ErrUnsupportedOS }, stubVersion)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"install"})

	err := cmd.Execute()

	require.Error(t, err)
	require.True(t, errors.Is(err, ErrUnsupportedOS))
}

func TestInstallCmd_WritesDefaultConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	stub := &stubBackend{}
	buf, run := makeCmd(t, stub, stubVersion)

	err := run("install")

	require.NoError(t, err)
	require.Contains(t, buf.String(), "Default config written to")
}

func TestInstallCmd_SkipsExistingConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	_, err := WriteDefaultConfigIfAbsent()
	require.NoError(t, err)

	stub := &stubBackend{}
	buf, run := makeCmd(t, stub, stubVersion)

	err = run("install")

	require.NoError(t, err)
	require.Contains(t, buf.String(), "was not modified")
}

func TestUninstallCmd_PrintsConfirmation(t *testing.T) {
	stub := &stubBackend{}
	buf, run := makeCmd(t, stub, stubVersion)

	err := run("uninstall")

	require.NoError(t, err)
	require.True(t, stub.uninstalled)
	require.Contains(t, buf.String(), "Autorun uninstalled")
}

func TestUninstallCmd_BackendError(t *testing.T) {
	stub := &stubBackend{uninstallErr: errUnloadFailed}
	_, run := makeCmd(t, stub, stubVersion)

	err := run("uninstall")

	require.Error(t, err)
	require.Contains(t, err.Error(), "unload failed")
}
