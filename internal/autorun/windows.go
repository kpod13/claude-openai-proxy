//go:build windows

package autorun

import (
	"context"
	"fmt"

	"golang.org/x/sys/windows/registry"
)

const (
	windowsRunKey   = `Software\Microsoft\Windows\CurrentVersion\Run`
	windowsValueKey = "claude-openai-proxy"
)

type windowsBackend struct{}

func newWindowsBackend() Backend {
	return &windowsBackend{}
}

func (b *windowsBackend) Install(_ context.Context, cfg InstallConfig) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, windowsRunKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("autorun: open registry key: %w", err)
	}

	defer func() { _ = k.Close() }()

	err = k.SetStringValue(windowsValueKey, cfg.BinaryPath)
	if err != nil {
		return fmt.Errorf("autorun: write registry value: %w", err)
	}

	return nil
}

func (b *windowsBackend) Uninstall(_ context.Context) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, windowsRunKey, registry.SET_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil
		}

		return fmt.Errorf("autorun: open registry key: %w", err)
	}

	defer func() { _ = k.Close() }()

	err = k.DeleteValue(windowsValueKey)
	if err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("autorun: delete registry value: %w", err)
	}

	return nil
}
