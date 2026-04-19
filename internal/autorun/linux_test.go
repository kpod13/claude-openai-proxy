package autorun

import (
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
	require.Contains(t, string(content), "Exec=/usr/local/bin/claude-openai-proxy")
	require.Contains(t, string(content), "X-GNOME-Autostart-enabled=true")
}
