//go:build !windows

package autorun

import (
	"context"
	"errors"
)

var (
	errWindowsOnly = errors.New("autorun: Windows backend is only available on Windows")
)

type windowsBackend struct{}

func newWindowsBackend() Backend {
	return &windowsBackend{}
}

func (b *windowsBackend) Install(_ context.Context, _ InstallConfig) error {
	return errWindowsOnly
}

func (b *windowsBackend) Uninstall(_ context.Context) error {
	return errWindowsOnly
}
