package autorun

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteDefaultConfigIfAbsent_CreatesFile(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HOME", dir)

	created, err := WriteDefaultConfigIfAbsent()

	require.NoError(t, err)
	require.True(t, created)

	content, err := os.ReadFile(filepath.Join(dir, defaultConfigName))
	require.NoError(t, err)
	require.Contains(t, string(content), "listen:")
	require.Contains(t, string(content), "127.0.0.1:8080")
}

func TestWriteDefaultConfigIfAbsent_SkipsExistingFile(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("HOME", dir)

	original := []byte("listen: \"0.0.0.0:9090\"\n")
	path := filepath.Join(dir, defaultConfigName)

	err := os.WriteFile(path, original, 0o600)
	require.NoError(t, err)

	created, err := WriteDefaultConfigIfAbsent()

	require.NoError(t, err)
	require.False(t, created)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, original, content)
}
