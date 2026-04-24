package autorun

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strconv"
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
// html/template is used to safely escape XML special characters in paths.
func generatePlist(cfg InstallConfig) ([]byte, error) {
	var buf bytes.Buffer

	err := plistTmpl.Execute(&buf, cfg)
	if err != nil {
		return nil, fmt.Errorf("autorun: render plist: %w", err)
	}

	return buf.Bytes(), nil
}

// launchctlTarget returns the launchd bootstrap target for the current user.
func launchctlTarget() string {
	return "gui/" + strconv.Itoa(os.Getuid())
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

	out, err := execCommand(ctx, "launchctl", "bootstrap", launchctlTarget(), path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("autorun: launchctl bootstrap: %w\n%s", err, out)
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

	out, err := execCommand(ctx, "launchctl", "bootout", launchctlTarget(), path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("autorun: launchctl bootout: %w\n%s", err, out)
	}

	err = os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("autorun: remove plist: %w", err)
	}

	return nil
}
