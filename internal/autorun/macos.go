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
	plistServiceName = "com.claude-openai-proxy"
)

var (
	plistTmpl = template.Must(template.New("plist").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>{{ .Label }}</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{ .BinaryPath }}</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<false/>
</dict>
</plist>
`))
)

type macosBackend struct{}

func newMacOSBackend() Backend {
	return &macosBackend{}
}

func (b *macosBackend) plistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("autorun: get home dir: %w", err)
	}

	return filepath.Join(home, "Library", "LaunchAgents", plistServiceName+".plist"), nil
}

// generatePlist renders the launchd plist XML for the given config.
func generatePlist(cfg InstallConfig) ([]byte, error) {
	var buf bytes.Buffer

	err := plistTmpl.Execute(&buf, cfg)
	if err != nil {
		return nil, fmt.Errorf("autorun: render plist: %w", err)
	}

	return buf.Bytes(), nil
}

func (b *macosBackend) Install(ctx context.Context, cfg InstallConfig) error {
	path, err := b.plistPath()
	if err != nil {
		return err
	}

	data, err := generatePlist(cfg)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(path), 0o750)
	if err != nil {
		return fmt.Errorf("autorun: create LaunchAgents dir: %w", err)
	}

	err = os.WriteFile(path, data, 0o600)
	if err != nil {
		return fmt.Errorf("autorun: write plist: %w", err)
	}

	out, err := exec.CommandContext(ctx, "launchctl", "load", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("autorun: launchctl load: %w\n%s", err, out)
	}

	return nil
}

func (b *macosBackend) Uninstall(ctx context.Context) error {
	path, err := b.plistPath()
	if err != nil {
		return err
	}

	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("autorun: stat plist: %w", err)
	}

	out, err := exec.CommandContext(ctx, "launchctl", "unload", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("autorun: launchctl unload: %w\n%s", err, out)
	}

	err = os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("autorun: remove plist: %w", err)
	}

	return nil
}
