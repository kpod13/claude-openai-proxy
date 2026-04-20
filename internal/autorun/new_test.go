package autorun

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew_CurrentPlatformSucceeds(t *testing.T) {
	backend, err := New()

	require.NoError(t, err)
	require.NotNil(t, backend)
}

func TestNew_UnsupportedOSError(t *testing.T) {
	err := fmt.Errorf("%w: %s", ErrUnsupportedOS, "plan9")

	require.Error(t, err)
	require.True(t, errors.Is(err, ErrUnsupportedOS))
	require.Contains(t, err.Error(), "plan9")
}

func TestErrUnsupportedOS_IsSentinel(t *testing.T) {
	require.NotNil(t, ErrUnsupportedOS)
}
