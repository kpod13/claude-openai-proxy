package autorun

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinuxBackend_SystemdUnitContent(t *testing.T) {
	cfg := InstallConfig{
		BinaryPath: "/usr/local/bin/claude-openai-proxy",
		Label:      "Claude OpenAI Proxy",
	}

	content, err := generateSystemdUnit(cfg)

	require.NoError(t, err)
	assert.Contains(t, string(content), "Description=Claude OpenAI Proxy")
	assert.Contains(t, string(content), "ExecStart=/usr/local/bin/claude-openai-proxy")
	assert.Contains(t, string(content), "[Install]")
	assert.Contains(t, string(content), "WantedBy=default.target")
}

func TestLinuxBackend_XDGDesktopContent(t *testing.T) {
	cfg := InstallConfig{
		BinaryPath: "/usr/local/bin/claude-openai-proxy",
		Label:      "claude-openai-proxy",
	}

	content, err := generateXDGDesktop(cfg)

	require.NoError(t, err)
	assert.Contains(t, string(content), "Type=Application")
	assert.Contains(t, string(content), "Exec=/usr/local/bin/claude-openai-proxy")
	assert.Contains(t, string(content), "X-GNOME-Autostart-enabled=true")
}
