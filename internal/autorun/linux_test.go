package autorun

import (
	"context"
	"os"
	"os/exec"
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

func TestLinuxBackend_SystemdAvailable(t *testing.T) {
	cases := []struct {
		name     string
		lookPath string
		cmd      func(context.Context, string, ...string) *exec.Cmd
		want     bool
	}{
		{
			name:     "lookpath fails",
			lookPath: "",
			want:     false,
		},
		{
			name:     "running",
			lookPath: "/usr/bin/systemctl",
			cmd:      cmdSuccess("running"),
			want:     true,
		},
		{
			name:     "degraded",
			lookPath: "/usr/bin/systemctl",
			cmd:      cmdFail("degraded"),
			want:     true,
		},
		{
			name:     "stopped",
			lookPath: "/usr/bin/systemctl",
			cmd:      cmdFail("stopped"),
			want:     false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockExec(t, tc.lookPath, tc.cmd)

			b := &linuxBackend{}

			require.Equal(t, tc.want, b.systemdAvailable(context.Background()))
		})
	}
}

// --- installSystemd ---

func TestLinuxBackend_InstallSystemd(t *testing.T) {
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
				mockExec(t, "/usr/bin/systemctl", cmdSuccess(""))
			},
			check: func(t *testing.T, dir string) {
				t.Helper()

				unitPath := filepath.Join(dir, ".config", "systemd", "user", linuxUnitName)
				_, statErr := os.Stat(unitPath)
				require.NoError(t, statErr)
			},
		},
		{
			name: "command fails",
			setup: func(t *testing.T, _ string) {
				t.Helper()
				mockExec(t, "/usr/bin/systemctl", cmdFail(""))
			},
			wantErr:     true,
			errContains: "systemctl enable",
		},
		{
			name: "home dir error",
			setup: func(t *testing.T, _ string) {
				t.Helper()
				t.Setenv("HOME", fileAsHome(t))
				mockExec(t, "/usr/bin/systemctl", cmdSuccess(""))
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("HOME", dir)
			tc.setup(t, dir)

			b := &linuxBackend{}
			cfg := InstallConfig{BinaryPath: "/bin/proxy", Label: "test"}

			err := b.installSystemd(context.Background(), cfg)

			if tc.wantErr {
				require.Error(t, err)

				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}

				return
			}

			require.NoError(t, err)

			if tc.check != nil {
				tc.check(t, dir)
			}
		})
	}
}

// --- installXDG ---

func TestLinuxBackend_InstallXDG(t *testing.T) {
	cases := []struct {
		name        string
		setup       func(t *testing.T, dir string)
		wantErr     bool
		check       func(t *testing.T, dir string)
	}{
		{
			name: "success",
			check: func(t *testing.T, dir string) {
				t.Helper()

				desktopPath := filepath.Join(dir, ".config", "autostart", linuxDesktopName)
				_, statErr := os.Stat(desktopPath)
				require.NoError(t, statErr)
			},
		},
		{
			name: "home dir error",
			setup: func(t *testing.T, _ string) {
				t.Helper()
				t.Setenv("HOME", fileAsHome(t))
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("HOME", dir)

			if tc.setup != nil {
				tc.setup(t, dir)
			}

			b := &linuxBackend{}
			cfg := InstallConfig{BinaryPath: "/bin/proxy", Label: "test"}

			err := b.installXDG(cfg)

			if tc.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)

			if tc.check != nil {
				tc.check(t, dir)
			}
		})
	}
}

// --- Install ---

func TestLinuxBackend_Install(t *testing.T) {
	cases := []struct {
		name     string
		lookPath string
	}{
		{
			name:     "via systemd",
			lookPath: "/usr/bin/systemctl",
		},
		{
			// LookPath fails → systemdAvailable returns false → falls through to XDG.
			name:     "via xdg",
			lookPath: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("HOME", dir)
			mockExec(t, tc.lookPath, cmdSuccess(""))

			b := &linuxBackend{}
			cfg := InstallConfig{BinaryPath: "/bin/proxy", Label: "test"}

			err := b.Install(context.Background(), cfg)

			require.NoError(t, err)
		})
	}
}

// --- Uninstall ---

func TestLinuxBackend_Uninstall(t *testing.T) {
	cases := []struct {
		name        string
		setup       func(t *testing.T, dir string)
		wantErr     bool
		errContains string
		check       func(t *testing.T, dir string)
	}{
		{
			name: "no files",
		},
		{
			name: "removes desktop file",
			setup: func(t *testing.T, dir string) {
				t.Helper()

				desktopDir := filepath.Join(dir, ".config", "autostart")
				err := os.MkdirAll(desktopDir, 0o750)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(desktopDir, linuxDesktopName), []byte("[Desktop Entry]"), 0o600)
				require.NoError(t, err)
			},
			check: func(t *testing.T, dir string) {
				t.Helper()

				_, statErr := os.Stat(filepath.Join(dir, ".config", "autostart", linuxDesktopName))
				require.True(t, os.IsNotExist(statErr))
			},
		},
		{
			name: "XDG remove error",
			setup: func(t *testing.T, _ string) {
				t.Helper()
				t.Setenv("HOME", fileAsHome(t))
			},
			wantErr:     true,
			errContains: "remove XDG desktop",
		},
		{
			name: "systemd success",
			setup: func(t *testing.T, dir string) {
				t.Helper()

				mockExec(t, "/usr/bin/systemctl", cmdSuccess(""))

				unitDir := filepath.Join(dir, ".config", "systemd", "user")
				err := os.MkdirAll(unitDir, 0o750)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(unitDir, linuxUnitName), []byte("[Unit]"), 0o600)
				require.NoError(t, err)
			},
			check: func(t *testing.T, dir string) {
				t.Helper()

				_, statErr := os.Stat(filepath.Join(dir, ".config", "systemd", "user", linuxUnitName))
				require.True(t, os.IsNotExist(statErr))
			},
		},
		{
			name: "systemd command fails",
			setup: func(t *testing.T, dir string) {
				t.Helper()

				mockExec(t, "/usr/bin/systemctl", cmdFail(""))

				unitDir := filepath.Join(dir, ".config", "systemd", "user")
				err := os.MkdirAll(unitDir, 0o750)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(unitDir, linuxUnitName), []byte("[Unit]"), 0o600)
				require.NoError(t, err)
			},
			wantErr:     true,
			errContains: "systemctl disable",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("HOME", dir)

			if tc.setup != nil {
				tc.setup(t, dir)
			}

			b := &linuxBackend{}

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
