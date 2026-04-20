package autorun

import (
	"errors"
	"fmt"
	"runtime"
)

var (
	// ErrUnsupportedOS is returned by New when the current OS has no autorun backend.
	ErrUnsupportedOS = errors.New("autorun: unsupported OS")
)

// New returns the Backend appropriate for the current operating system.
// It returns ErrUnsupportedOS (wrapped) for unknown GOOS values.
func New() (Backend, error) {
	switch runtime.GOOS {
	case "darwin":
		return newMacOSBackend(), nil
	case "linux":
		return newLinuxBackend(), nil
	case "windows":
		return newWindowsBackend(), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedOS, runtime.GOOS)
	}
}
