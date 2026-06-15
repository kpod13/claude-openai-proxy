package autorun

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

const (
	plistServiceName = "com.claude-openai-proxy"

	// launchAgentPath is the PATH provided to the launchd agent. launchd hands
	// jobs a minimal PATH (/usr/bin:/bin:/usr/sbin:/sbin) that lacks the claude
	// CLI, so the server would fail model discovery on launch; include the
	// common Homebrew and system locations so it can find claude.
	launchAgentPath = "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"
)

var (
	plistTmpl = template.Must(template.New("plist").Funcs(template.FuncMap{"xml": xmlEscape}).Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>{{ xml .Label }}</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{ xml .BinaryPath }}</string>
	</array>
	<key>EnvironmentVariables</key>
	<dict>
		<key>PATH</key>
		<string>` + launchAgentPath + `</string>
	</dict>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
</dict>
</plist>
`))
)

// xmlEscape escapes XML special characters in user-controlled values
// (binary path, label) for safe inclusion in the plist.
func xmlEscape(s string) string {
	var b strings.Builder
	// xml.EscapeText only fails if the writer fails; strings.Builder never does.
	_ = xml.EscapeText(&b, []byte(s))

	return b.String()
}

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
// text/template is used (not html/template, which mangles the <?xml ...?>
// declaration); user-controlled values are escaped via the xml func.
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
		// Keep install atomic: don't leave a half-installed plist behind.
		_ = os.Remove(path)

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
