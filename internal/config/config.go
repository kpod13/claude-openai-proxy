package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// RateLimit holds optional rate limiting configuration.
// Zero values for both fields mean rate limiting is disabled.
type RateLimit struct {
	RequestsPerMinute int `yaml:"requests_per_minute"`
	TokensPerMinute   int `yaml:"tokens_per_minute"`
}

// Permission mode values accepted by the claude CLI --permission-mode flag.
// These are the only valid values for Permission.Mode.
const (
	// ModeDefault asks for permission as usual. Headless (no TTY) tool calls
	// that need approval will hang, so this is the safe, behavior-preserving
	// default.
	ModeDefault = "default"
	// ModeAcceptEdits auto-accepts file edits but still asks for other tools.
	ModeAcceptEdits = "acceptEdits"
	// ModePlan is planning mode; no tools are executed.
	ModePlan = "plan"
	// ModeDontAsk does not prompt and relies on the allowed/disallowed lists.
	ModeDontAsk = "dontAsk"
	// ModeAuto lets claude decide automatically.
	ModeAuto = "auto"
	// ModeBypassPermissions skips all permission checks. DANGER: this is
	// effectively unauthenticated remote code execution for anyone who can
	// reach the listener.
	ModeBypassPermissions = "bypassPermissions"
)

// Permission holds the optional claude permission policy applied to every
// claude invocation. The safe default (Mode ModeDefault, empty lists)
// allowlists no tools and bypasses no permission checks, so headless behavior
// is unchanged.
//
// Mode must be one of the Mode* constants. Each AllowedTools / DisallowedTools
// entry is a claude tool spec — ToolName or ToolName(rule), e.g. "Write",
// "Bash(git *)", "mcp__server__tool". AddDirs entries are directory paths.
type Permission struct {
	Mode            string   `yaml:"mode"`
	AllowedTools    []string `yaml:"allowed_tools"`
	DisallowedTools []string `yaml:"disallowed_tools"`
	AddDirs         []string `yaml:"add_dirs"`
}

var (
	// validPermissionModes is the set of accepted Permission.Mode values,
	// derived from the Mode* constants so validation and documentation stay
	// in sync.
	validPermissionModes = map[string]bool{
		ModeDefault:           true,
		ModeAcceptEdits:       true,
		ModePlan:              true,
		ModeDontAsk:           true,
		ModeAuto:              true,
		ModeBypassPermissions: true,
	}

	// toolSpecRe matches a claude tool spec: a tool name (built-in like Bash/Edit
	// or MCP like mcp__server__tool) optionally followed by a parenthesized rule,
	// e.g. "Write", "Bash(git *)", "WebFetch(domain:example.com)".
	toolSpecRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\([^)]+\))?$`)

	// errInvalidMode is returned when permission.mode is not a supported value.
	errInvalidMode = errors.New("config: invalid permission mode " +
		"(supported: acceptEdits, auto, bypassPermissions, default, dontAsk, plan)")

	// errBlankEntry is returned for an empty or whitespace-only list entry.
	errBlankEntry = errors.New("config: permission list entry must not be blank")

	// errFlagLikeEntry is returned for a list entry that begins with '-'.
	errFlagLikeEntry = errors.New("config: permission list entry must not begin with '-'")

	// errInvalidToolSpec is returned for a malformed tool spec.
	errInvalidToolSpec = errors.New("config: invalid permission tool spec " +
		"(expected ToolName or ToolName(rule))")
)

// Config holds server configuration loaded from a YAML file.
type Config struct {
	Listen     string     `yaml:"listen"`
	Aliases    []string   `yaml:"aliases"`
	RateLimit  RateLimit  `yaml:"rate_limit"`
	Permission Permission `yaml:"permission"`
}

// defaultConfig returns built-in defaults used when no config file is found.
func defaultConfig() Config {
	return Config{
		Listen:     "127.0.0.1:8080",
		Aliases:    []string{"opus", "sonnet", "haiku"},
		Permission: Permission{Mode: ModeDefault},
	}
}

// Load loads configuration from the given explicit path, or searches standard
// locations when path is empty. Returns built-in defaults if no file is found.
//
// Search order (first match wins):
//  1. explicit path (error if given but missing)
//  2. ~/.claude-code-openai-server.yaml
//  3. /etc/claude-code-openai-server/config.yaml
//  4. built-in defaults
func Load(path string) (*Config, error) {
	if path != "" {
		return loadFile(path)
	}

	for _, p := range searchPaths() {
		cfg, err := loadFile(p)
		if err == nil {
			return cfg, nil
		}

		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	def := defaultConfig()

	return &def, nil
}

// searchPathsFn is the function used to get search paths; replaced in tests.
var (
	searchPathsFn = defaultSearchPaths
)

// searchPaths delegates to searchPathsFn so tests can inject custom paths.
func searchPaths() []string { return searchPathsFn() }

// defaultSearchPaths returns the standard config file locations in priority order.
func defaultSearchPaths() []string {
	paths := []string{"/etc/claude-code-openai-server/config.yaml"}

	home, err := os.UserHomeDir()
	if err == nil {
		paths = append([]string{filepath.Join(home, ".claude-code-openai-server.yaml")}, paths...)
	}

	return paths
}

// loadFile reads and decodes a YAML config file, merging over built-in defaults.
func loadFile(path string) (*Config, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("config: open %s: %w", path, err)
	}

	defer func() { _ = f.Close() }()

	cfg := defaultConfig()

	err = yaml.NewDecoder(f).Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	err = cfg.validate()
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validate checks the permission policy and normalizes its entries in place.
// It is called at load time so an invalid policy fails startup before the
// server binds, rather than failing every request later.
func (c *Config) validate() error {
	p := &c.Permission

	if p.Mode == "" {
		p.Mode = ModeDefault
	}

	if !validPermissionModes[p.Mode] {
		return fmt.Errorf("%w: %q", errInvalidMode, p.Mode)
	}

	err := validateToolSpecs("allowed_tools", p.AllowedTools)
	if err != nil {
		return err
	}

	err = validateToolSpecs("disallowed_tools", p.DisallowedTools)
	if err != nil {
		return err
	}

	return validateAddDirs(p.AddDirs)
}

// validateToolSpecs validates and trims each tool spec in place.
func validateToolSpecs(field string, specs []string) error {
	for i, raw := range specs {
		spec := strings.TrimSpace(raw)

		switch {
		case spec == "":
			return fmt.Errorf("%w (permission.%s)", errBlankEntry, field)
		case strings.HasPrefix(spec, "-"):
			return fmt.Errorf("%w: permission.%s %q", errFlagLikeEntry, field, spec)
		case !toolSpecRe.MatchString(spec):
			return fmt.Errorf("%w: permission.%s %q", errInvalidToolSpec, field, spec)
		}

		specs[i] = spec
	}

	return nil
}

// validateAddDirs validates and trims each directory entry in place.
func validateAddDirs(dirs []string) error {
	for i, raw := range dirs {
		dir := strings.TrimSpace(raw)

		switch {
		case dir == "":
			return fmt.Errorf("%w (permission.add_dirs)", errBlankEntry)
		case strings.HasPrefix(dir, "-"):
			return fmt.Errorf("%w: permission.add_dirs %q", errFlagLikeEntry, dir)
		}

		dirs[i] = dir
	}

	return nil
}
