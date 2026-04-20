// Package autorun provides user-level autostart provisioning for the proxy binary.
// It supports macOS (launchd), Linux (systemd user / XDG), and Windows (registry).
// Other operating systems return ErrUnsupportedOS.
package autorun

import "context"

// InstallConfig carries the parameters needed to create an autostart entry.
type InstallConfig struct {
	// BinaryPath is the absolute path to the proxy binary (from os.Executable).
	BinaryPath string
	// Label is the human-readable name used in the autostart entry.
	Label string
}

// Backend abstracts OS-specific autostart mechanisms.
type Backend interface {
	// Install registers the binary as a user-level autostart entry.
	Install(ctx context.Context, cfg InstallConfig) error
	// Uninstall removes the autostart entry. It is idempotent: if no entry
	// exists it returns nil.
	Uninstall(ctx context.Context) error
}
