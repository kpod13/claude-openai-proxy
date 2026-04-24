package autorun

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLinuxBackend_SystemdUnitContent(t *testing.T) {
	cfg := InstallConfig{
		BinaryPath: "/usr/local/bin/claude-openai-proxy",
		Label:      "Claude OpenAI Proxy",
	}

	content, err := generateSystemdUnit(cfg)

	require.NoError(t, err)
	require.Contains(t, string(content), "Description=Claude OpenAI Proxy")
	require.Contains(t, string(content), "ExecStart=/usr/local/bin/claude-openai-proxy")
	require.Contains(t, string(content), "[Install]")
	require.Contains(t, string(content), "WantedBy=default.target")
}

func TestLinuxBackend_XDGDesktopContent(t *testing.T) {
	cfg := InstallConfig{
		BinaryPath: "/usr/local/bin/claude-openai-proxy",
		Label:      "claude-openai-proxy",
	}

	content, err := generateXDGDesktop(cfg)

	require.NoError(t, err)
	require.Contains(t, string(content), "Type=Application")
	require.Contains(t, string(content), `Exec="/usr/local/bin/claude-openai-proxy"`)
	require.Contains(t, string(content), "X-GNOME-Autostart-enabled=true")
}

func TestNewLinuxBackend(t *testing.T) {
	b := newLinuxBackend()

	require.NotNil(t, b)
}

func TestLinuxBackend_SystemdUnitPath(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HOME", dir)

	b := &linuxBackend{}

	path, err := b.systemdUnitPath()

	require.NoError(t, err)
	require.Contains(t, path, "systemd")
	require.Contains(t, path, linuxUnitName)
}

func TestLinuxBackend_XDGDesktopPath(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HOME", dir)

	b := &linuxBackend{}

	path, err := b.xdgDesktopPath()

	require.NoError(t, err)
	require.Contains(t, path, "autostart")
	require.Contains(t, path, linuxDesktopName)
}

// --- systemdAvailable ---

func TestLinuxBackend_SystemdAvailable_LookPathFails(t *testing.T) {
	mockExec(t, "", nil)

	b := &linuxBackend{}

	require.False(t, b.systemdAvailable(context.Background()))
}

func TestLinuxBackend_SystemdAvailable_Running(t *testing.T) {
	mockExec(t, "/usr/bin/systemctl", cmdSuccess("running"))

	b := &linuxBackend{}

	require.True(t, b.systemdAvailable(context.Background()))
}

func TestLinuxBackend_SystemdAvailable_Degraded(t *testing.T) {
	// systemd is running but some units have failed — still usable.
	mockExec(t, "/usr/bin/systemctl", cmdFail("degraded"))

	b := &linuxBackend{}

	require.True(t, b.systemdAvailable(context.Background()))
}

func TestLinuxBackend_SystemdAvailable_Stopped(t *testing.T) {
	mockExec(t, "/usr/bin/systemctl", cmdFail("stopped"))

	b := &linuxBackend{}

	require.False(t, b.systemdAvailable(context.Background()))
}

// --- installSystemd ---

func TestLinuxBackend_InstallSystemd_Success(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HOME", dir)
	mockExec(t, "/usr/bin/systemctl", cmdSuccess(""))

	b := &linuxBackend{}
	cfg := InstallConfig{BinaryPath: "/bin/proxy", Label: "test"}

	err := b.installSystemd(context.Background(), cfg)

	require.NoError(t, err)

	unitPath := filepath.Join(dir, ".config", "systemd", "user", linuxUnitName)
	_, statErr := os.Stat(unitPath)
	require.NoError(t, statErr)
}

func TestLinuxBackend_InstallSystemd_CommandFails(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HOME", dir)
	mockExec(t, "/usr/bin/systemctl", cmdFail(""))

	b := &linuxBackend{}
	cfg := InstallConfig{BinaryPath: "/bin/proxy", Label: "test"}

	err := b.installSystemd(context.Background(), cfg)

	require.Error(t, err)
	require.Contains(t, err.Error(), "systemctl enable")
}

func TestLinuxBackend_InstallSystemd_HomeDirError(t *testing.T) {
	t.Setenv("HOME", fileAsHome(t))
	mockExec(t, "/usr/bin/systemctl", cmdSuccess(""))

	b := &linuxBackend{}
	cfg := InstallConfig{BinaryPath: "/bin/proxy", Label: "test"}

	err := b.installSystemd(context.Background(), cfg)

	require.Error(t, err)
}

// --- installXDG ---

func TestLinuxBackend_InstallXDG_Success(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HOME", dir)

	b := &linuxBackend{}
	cfg := InstallConfig{BinaryPath: "/bin/proxy", Label: "test"}

	err := b.installXDG(cfg)

	require.NoError(t, err)

	desktopPath := filepath.Join(dir, ".config", "autostart", linuxDesktopName)
	_, statErr := os.Stat(desktopPath)
	require.NoError(t, statErr)
}

func TestLinuxBackend_InstallXDG_HomeDirError(t *testing.T) {
	t.Setenv("HOME", fileAsHome(t))

	b := &linuxBackend{}
	cfg := InstallConfig{BinaryPath: "/bin/proxy", Label: "test"}

	err := b.installXDG(cfg)

	require.Error(t, err)
}

// --- Install ---

func TestLinuxBackend_Install_ViaSystemd(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HOME", dir)
	mockExec(t, "/usr/bin/systemctl", cmdSuccess(""))

	b := &linuxBackend{}
	cfg := InstallConfig{BinaryPath: "/bin/proxy", Label: "test"}

	err := b.Install(context.Background(), cfg)

	require.NoError(t, err)
}

func TestLinuxBackend_Install_ViaXDG(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HOME", dir)
	// LookPath fails → systemdAvailable returns false → falls through to XDG.
	mockExec(t, "", nil)

	b := &linuxBackend{}
	cfg := InstallConfig{BinaryPath: "/bin/proxy", Label: "test"}

	err := b.Install(context.Background(), cfg)

	require.NoError(t, err)
}

// --- Uninstall ---

func TestLinuxBackend_Uninstall_NoFiles(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HOME", dir)

	b := &linuxBackend{}

	err := b.Uninstall(context.Background())

	require.NoError(t, err)
}

func TestLinuxBackend_Uninstall_RemovesDesktopFile(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HOME", dir)

	desktopDir := filepath.Join(dir, ".config", "autostart")
	err := os.MkdirAll(desktopDir, 0o750)
	require.NoError(t, err)

	desktopFile := filepath.Join(desktopDir, linuxDesktopName)
	err = os.WriteFile(desktopFile, []byte("[Desktop Entry]"), 0o600)
	require.NoError(t, err)

	b := &linuxBackend{}

	err = b.Uninstall(context.Background())

	require.NoError(t, err)

	_, statErr := os.Stat(desktopFile)
	require.True(t, os.IsNotExist(statErr))
}

func TestLinuxBackend_Uninstall_RemoveError(t *testing.T) {
	t.Setenv("HOME", fileAsHome(t))

	b := &linuxBackend{}

	err := b.Uninstall(context.Background())

	require.Error(t, err)
	require.Contains(t, err.Error(), "remove XDG desktop")
}

// --- uninstallSystemd ---

func TestLinuxBackend_UninstallSystemd_Success(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HOME", dir)

	unitDir := filepath.Join(dir, ".config", "systemd", "user")
	err := os.MkdirAll(unitDir, 0o750)
	require.NoError(t, err)

	unitPath := filepath.Join(unitDir, linuxUnitName)
	err = os.WriteFile(unitPath, []byte("[Unit]"), 0o600)
	require.NoError(t, err)

	mockExec(t, "/usr/bin/systemctl", cmdSuccess(""))

	b := &linuxBackend{}

	err = b.Uninstall(context.Background())

	require.NoError(t, err)

	_, statErr := os.Stat(unitPath)
	require.True(t, os.IsNotExist(statErr))
}

func TestLinuxBackend_UninstallSystemd_CommandFails(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HOME", dir)

	unitDir := filepath.Join(dir, ".config", "systemd", "user")
	err := os.MkdirAll(unitDir, 0o750)
	require.NoError(t, err)

	unitPath := filepath.Join(unitDir, linuxUnitName)
	err = os.WriteFile(unitPath, []byte("[Unit]"), 0o600)
	require.NoError(t, err)

	mockExec(t, "/usr/bin/systemctl", cmdFail(""))

	b := &linuxBackend{}

	err = b.Uninstall(context.Background())

	require.Error(t, err)
	require.Contains(t, err.Error(), "systemctl disable")
}
