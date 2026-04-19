package autorun

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

const (
	linuxServiceName = "claude-openai-proxy"
	linuxUnitName    = linuxServiceName + ".service"
	linuxDesktopName = linuxServiceName + ".desktop"
)

var (
	systemdTmpl = template.Must(template.New("systemd").Parse(`[Unit]
Description={{ .Label }}
After=network.target

[Service]
ExecStart={{ .BinaryPath }}
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`))

	xdgTmpl = template.Must(template.New("xdg").Parse(`[Desktop Entry]
Type=Application
Name={{ .Label }}
Exec={{ .BinaryPath }}
X-GNOME-Autostart-enabled=true
`))
)

type linuxBackend struct{}

func newLinuxBackend() Backend {
	return &linuxBackend{}
}

func (b *linuxBackend) systemdUnitPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("autorun: get home dir: %w", err)
	}

	return filepath.Join(home, ".config", "systemd", "user", linuxUnitName), nil
}

func (b *linuxBackend) xdgDesktopPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("autorun: get home dir: %w", err)
	}

	return filepath.Join(home, ".config", "autostart", linuxDesktopName), nil
}

func (b *linuxBackend) systemdAvailable(ctx context.Context) bool {
	out, err := exec.CommandContext(ctx, "systemctl", "--user", "is-system-running").CombinedOutput()
	if err != nil {
		_, statErr := exec.LookPath("systemctl")

		return statErr == nil && len(out) > 0
	}

	return true
}

func (b *linuxBackend) Install(ctx context.Context, cfg InstallConfig) error {
	if b.systemdAvailable(ctx) {
		return b.installSystemd(ctx, cfg)
	}

	return b.installXDG(cfg)
}

// generateSystemdUnit renders the systemd unit file content for the given config.
func generateSystemdUnit(cfg InstallConfig) ([]byte, error) {
	var buf bytes.Buffer

	err := systemdTmpl.Execute(&buf, cfg)
	if err != nil {
		return nil, fmt.Errorf("autorun: render systemd unit: %w", err)
	}

	return buf.Bytes(), nil
}

func (b *linuxBackend) installSystemd(ctx context.Context, cfg InstallConfig) error {
	path, err := b.systemdUnitPath()
	if err != nil {
		return err
	}

	data, err := generateSystemdUnit(cfg)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(path), 0o750)
	if err != nil {
		return fmt.Errorf("autorun: create systemd user dir: %w", err)
	}

	err = os.WriteFile(path, data, 0o600)
	if err != nil {
		return fmt.Errorf("autorun: write systemd unit: %w", err)
	}

	out, err := exec.CommandContext(ctx, "systemctl", "--user", "enable", "--now", linuxServiceName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("autorun: systemctl enable: %w\n%s", err, out)
	}

	return nil
}

// generateXDGDesktop renders the XDG autostart desktop file content.
func generateXDGDesktop(cfg InstallConfig) ([]byte, error) {
	var buf bytes.Buffer

	err := xdgTmpl.Execute(&buf, cfg)
	if err != nil {
		return nil, fmt.Errorf("autorun: render XDG desktop: %w", err)
	}

	return buf.Bytes(), nil
}

func (b *linuxBackend) installXDG(cfg InstallConfig) error {
	path, err := b.xdgDesktopPath()
	if err != nil {
		return err
	}

	data, err := generateXDGDesktop(cfg)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(path), 0o750)
	if err != nil {
		return fmt.Errorf("autorun: create autostart dir: %w", err)
	}

	err = os.WriteFile(path, data, 0o600)
	if err != nil {
		return fmt.Errorf("autorun: write XDG desktop: %w", err)
	}

	return nil
}

func (b *linuxBackend) Uninstall(ctx context.Context) error {
	unitPath, err := b.systemdUnitPath()
	if err != nil {
		return err
	}

	_, unitErr := os.Stat(unitPath)
	if unitErr == nil {
		return b.uninstallSystemd(ctx, unitPath)
	}

	desktopPath, err := b.xdgDesktopPath()
	if err != nil {
		return err
	}

	err = os.Remove(desktopPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("autorun: remove XDG desktop: %w", err)
	}

	return nil
}

func (b *linuxBackend) uninstallSystemd(ctx context.Context, unitPath string) error {
	out, err := exec.CommandContext(ctx, "systemctl", "--user", "disable", "--now", linuxServiceName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("autorun: systemctl disable: %w\n%s", err, out)
	}

	err = os.Remove(unitPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("autorun: remove systemd unit: %w", err)
	}

	return nil
}
