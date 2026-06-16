package autorun

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultConfigName = ".claude-code-openai-server.yaml"

	// defaultConfigContent is the YAML written when no config file exists yet.
	defaultConfigContent = `# claude-openai-proxy configuration
listen: "127.0.0.1:8080"
aliases:
  - opus
  - sonnet
  - haiku

# Permission policy applied to every headless claude invocation.
# The default below is the safest: no tools are allowlisted and no permission
# checks are bypassed, so behavior is identical to a build without this block.
# Uncomment and tighten to let headless tool calls succeed instead of hang.
permission:
  # mode controls how claude handles tool permissions. One of:
  #   default            ask for permission as usual (headless => tool calls that
  #                      need approval will hang; the safe default)
  #   acceptEdits        auto-accept file edits, still ask for other tools
  #   plan               planning mode; no tools are executed
  #   dontAsk            do not prompt (relies on allowed/disallowed lists)
  #   auto               let claude decide automatically
  #   bypassPermissions  DANGER: skip all permission checks (effectively RCE for
  #                      anyone who can reach the listener)
  mode: default
  # allowed_tools: tool specs allowed without prompting.
  # Format: ToolName or ToolName(rule), e.g. Write, Edit, "Bash(git *)",
  # "WebFetch(domain:example.com)", mcp__server__tool.
  allowed_tools: []
  # disallowed_tools: tool specs to deny, same format as allowed_tools.
  disallowed_tools: []
  # add_dirs: extra directories tools may access (absolute paths recommended).
  add_dirs: []
`
)

// WriteDefaultConfigIfAbsent writes a default YAML config to
// ~/.claude-code-openai-server.yaml only if that file does not already exist.
// It returns (true, nil) when the file was created and (false, nil) when the
// file already existed and was left unchanged.
func WriteDefaultConfigIfAbsent() (created bool, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Errorf("autorun: get home dir: %w", err)
	}

	path := filepath.Join(home, defaultConfigName)

	_, err = os.Stat(path)
	if err == nil {
		return false, nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("autorun: stat config: %w", err)
	}

	err = os.WriteFile(path, []byte(defaultConfigContent), 0o600)
	if err != nil {
		return false, fmt.Errorf("autorun: write default config: %w", err)
	}

	return true, nil
}
