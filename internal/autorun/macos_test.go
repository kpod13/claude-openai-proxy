package autorun

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// fileAsHome returns a path to a regular file that can be used as HOME to
// force ENOTDIR errors when code tries to stat or create paths underneath it.
func fileAsHome(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	fakeHome := filepath.Join(dir, "not_a_dir")

	err := os.WriteFile(fakeHome, []byte("x"), 0o600)
	require.NoError(t, err)

	return fakeHome
}

func TestMacOSBackend_PlistContent(t *testing.T) {
	cfg := InstallConfig{
		BinaryPath: "/usr/local/bin/claude-openai-proxy",
		Label:      "com.claude-openai-proxy",
	}

	content, err := generatePlist(cfg)

	require.NoError(t, err)
	require.Contains(t, string(content), "<string>com.claude-openai-proxy</string>")
	require.Contains(t, string(content), "<string>/usr/local/bin/claude-openai-proxy</string>")
	require.Contains(t, string(content), "<true/>")
}

func TestMacOSBackend_PlistContent_XMLEscaping(t *testing.T) {
	cfg := InstallConfig{
		BinaryPath: "/home/user/my apps & tools/claude-openai-proxy",
		Label:      "com.claude-openai-proxy",
	}

	content, err := generatePlist(cfg)

	require.NoError(t, err)
	require.Contains(t, string(content), "my apps &amp; tools")
	require.NotContains(t, string(content), "apps & tools")
}

func TestLaunchctlTarget(t *testing.T) {
	target := launchctlTarget()

	require.Contains(t, target, "gui/")
}

// --- Install ---

func TestMacOSBackend_Install_Success(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HOME", dir)
	mockExec(t, "/usr/bin/launchctl", cmdSuccess(""))

	b := &macosBackend{}
	cfg := InstallConfig{BinaryPath: "/bin/proxy", Label: "test"}

	err := b.Install(context.Background(), cfg)

	require.NoError(t, err)

	plistFile := filepath.Join(dir, "Library", "LaunchAgents", plistServiceName+".plist")
	_, statErr := os.Stat(plistFile)
	require.NoError(t, statErr)
}

func TestMacOSBackend_Install_LaunchctlFails(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HOME", dir)
	mockExec(t, "/usr/bin/launchctl", cmdFail(""))

	b := &macosBackend{}
	cfg := InstallConfig{BinaryPath: "/bin/proxy", Label: "test"}

	err := b.Install(context.Background(), cfg)

	require.Error(t, err)
	require.Contains(t, err.Error(), "launchctl bootstrap")
}

func TestMacOSBackend_Install_MkdirFails(t *testing.T) {
	t.Setenv("HOME", fileAsHome(t))

	b := &macosBackend{}
	cfg := InstallConfig{BinaryPath: "/bin/proxy", Label: "test"}

	err := b.Install(context.Background(), cfg)

	require.Error(t, err)
	require.Contains(t, err.Error(), "create LaunchAgents dir")
}

// --- Uninstall ---

func TestMacOSBackend_Uninstall_Idempotent(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HOME", dir)

	b := &macosBackend{}

	err := b.Uninstall(context.Background())

	require.NoError(t, err)
}

func TestMacOSBackend_Uninstall_Success(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HOME", dir)
	mockExec(t, "/usr/bin/launchctl", cmdSuccess(""))

	agentsDir := filepath.Join(dir, "Library", "LaunchAgents")
	err := os.MkdirAll(agentsDir, 0o750)
	require.NoError(t, err)

	plistFile := filepath.Join(agentsDir, plistServiceName+".plist")
	err = os.WriteFile(plistFile, []byte("<plist/>"), 0o600)
	require.NoError(t, err)

	b := &macosBackend{}

	err = b.Uninstall(context.Background())

	require.NoError(t, err)

	_, statErr := os.Stat(plistFile)
	require.True(t, os.IsNotExist(statErr))
}

func TestMacOSBackend_Uninstall_LaunchctlFails(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HOME", dir)
	mockExec(t, "/usr/bin/launchctl", cmdFail(""))

	agentsDir := filepath.Join(dir, "Library", "LaunchAgents")
	err := os.MkdirAll(agentsDir, 0o750)
	require.NoError(t, err)

	plistFile := filepath.Join(agentsDir, plistServiceName+".plist")
	err = os.WriteFile(plistFile, []byte("<plist/>"), 0o600)
	require.NoError(t, err)

	b := &macosBackend{}

	err = b.Uninstall(context.Background())

	require.Error(t, err)
	require.Contains(t, err.Error(), "launchctl bootout")
}

func TestMacOSBackend_Uninstall_StatError(t *testing.T) {
	t.Setenv("HOME", fileAsHome(t))

	b := &macosBackend{}

	err := b.Uninstall(context.Background())

	require.Error(t, err)
	require.Contains(t, err.Error(), "stat plist")
}
