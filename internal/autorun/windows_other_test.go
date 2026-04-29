//go:build !windows

package autorun

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWindowsBackend_ErrorsOnNonWindows(t *testing.T) {
	b := newWindowsBackend()

	require.NotNil(t, b)

	err := b.Install(context.Background(), InstallConfig{})

	require.ErrorIs(t, err, errWindowsOnly)

	err = b.Uninstall(context.Background())

	require.ErrorIs(t, err, errWindowsOnly)
}
