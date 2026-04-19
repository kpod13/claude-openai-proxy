package autorun

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMacOSBackend_PlistContent(t *testing.T) {
	cfg := InstallConfig{
		BinaryPath: "/usr/local/bin/claude-openai-proxy",
		Label:      "com.claude-openai-proxy",
	}

	content, err := generatePlist(cfg)

	require.NoError(t, err)
	assert.Contains(t, string(content), "<string>com.claude-openai-proxy</string>")
	assert.Contains(t, string(content), "<string>/usr/local/bin/claude-openai-proxy</string>")
	assert.Contains(t, string(content), "<true/>")
}

func TestMacOSBackend_Uninstall_Idempotent(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HOME", dir)

	b := &macosBackend{}

	err := b.Uninstall(context.Background())
	assert.NoError(t, err)

	_ = os.Remove(dir) // no-op, just silence unused warning
}
