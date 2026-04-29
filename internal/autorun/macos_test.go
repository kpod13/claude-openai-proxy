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
	t.Parallel()

	cases := []struct {
		name            string
		cfg             InstallConfig
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "basic",
			cfg: InstallConfig{
				BinaryPath: "/usr/local/bin/claude-openai-proxy",
				Label:      "com.claude-openai-proxy",
			},
			wantContains: []string{
				"<string>com.claude-openai-proxy</string>",
				"<string>/usr/local/bin/claude-openai-proxy</string>",
				"<true/>",
			},
		},
		{
			name: "xml escaping",
			cfg: InstallConfig{
				BinaryPath: "/home/user/my apps & tools/claude-openai-proxy",
				Label:      "com.claude-openai-proxy",
			},
			wantContains:    []string{"my apps &amp; tools"},
			wantNotContains: []string{"apps & tools"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			content, err := generatePlist(tc.cfg)

			require.NoError(t, err)

			for _, want := range tc.wantContains {
				require.Contains(t, string(content), want)
			}

			for _, notWant := range tc.wantNotContains {
				require.NotContains(t, string(content), notWant)
			}
		})
	}
}

func TestLaunchctlTarget(t *testing.T) {
	target := launchctlTarget()

	require.Contains(t, target, "gui/")
}

// --- Install ---

func TestMacOSBackend_Install(t *testing.T) {
	cases := []struct {
		name        string
		setup       func(t *testing.T, dir string)
		wantErr     bool
		errContains string
		check       func(t *testing.T, dir string)
	}{
		{
			name: "success",
			setup: func(t *testing.T, _ string) {
				t.Helper()
				mockExec(t, "/usr/bin/launchctl", cmdSuccess(""))
			},
			check: func(t *testing.T, dir string) {
				t.Helper()

				plistFile := filepath.Join(dir, "Library", "LaunchAgents", plistServiceName+".plist")
				_, statErr := os.Stat(plistFile)
				require.NoError(t, statErr)
			},
		},
		{
			name: "launchctl fails",
			setup: func(t *testing.T, _ string) {
				t.Helper()
				mockExec(t, "/usr/bin/launchctl", cmdFail(""))
			},
			wantErr:     true,
			errContains: "launchctl bootstrap",
		},
		{
			name: "mkdir fails",
			setup: func(t *testing.T, _ string) {
				t.Helper()
				t.Setenv("HOME", fileAsHome(t))
			},
			wantErr:     true,
			errContains: "create LaunchAgents dir",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("HOME", dir)
			tc.setup(t, dir)

			b := &macosBackend{}
			cfg := InstallConfig{BinaryPath: "/bin/proxy", Label: "test"}

			err := b.Install(context.Background(), cfg)

			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)

				return
			}

			require.NoError(t, err)

			if tc.check != nil {
				tc.check(t, os.Getenv("HOME"))
			}
		})
	}
}

// --- Uninstall ---

func TestMacOSBackend_Uninstall(t *testing.T) {
	cases := []struct {
		name        string
		setup       func(t *testing.T, dir string)
		wantErr     bool
		errContains string
		check       func(t *testing.T, dir string)
	}{
		{
			name: "idempotent when no plist",
		},
		{
			name: "success",
			setup: func(t *testing.T, dir string) {
				t.Helper()

				mockExec(t, "/usr/bin/launchctl", cmdSuccess(""))

				agentsDir := filepath.Join(dir, "Library", "LaunchAgents")
				err := os.MkdirAll(agentsDir, 0o750)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(agentsDir, plistServiceName+".plist"), []byte("<plist/>"), 0o600)
				require.NoError(t, err)
			},
			check: func(t *testing.T, dir string) {
				t.Helper()

				plistFile := filepath.Join(dir, "Library", "LaunchAgents", plistServiceName+".plist")
				_, statErr := os.Stat(plistFile)
				require.True(t, os.IsNotExist(statErr))
			},
		},
		{
			name: "launchctl fails",
			setup: func(t *testing.T, dir string) {
				t.Helper()

				mockExec(t, "/usr/bin/launchctl", cmdFail(""))

				agentsDir := filepath.Join(dir, "Library", "LaunchAgents")
				err := os.MkdirAll(agentsDir, 0o750)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(agentsDir, plistServiceName+".plist"), []byte("<plist/>"), 0o600)
				require.NoError(t, err)
			},
			wantErr:     true,
			errContains: "launchctl bootout",
		},
		{
			name: "stat error",
			setup: func(t *testing.T, _ string) {
				t.Helper()
				t.Setenv("HOME", fileAsHome(t))
			},
			wantErr:     true,
			errContains: "stat plist",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("HOME", dir)

			if tc.setup != nil {
				tc.setup(t, dir)
			}

			b := &macosBackend{}

			err := b.Uninstall(context.Background())

			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)

				return
			}

			require.NoError(t, err)

			if tc.check != nil {
				tc.check(t, os.Getenv("HOME"))
			}
		})
	}
}
