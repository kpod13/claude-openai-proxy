package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad_Permission(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name    string
		content string
		wantErr bool
		check   func(t *testing.T, cfg *Config)
	}

	// Supported modes are validated as individual rows appended below.
	modes := []string{
		ModeDefault, ModeAcceptEdits, ModePlan, ModeDontAsk, ModeAuto, ModeBypassPermissions,
	}

	static := []testCase{
		{
			name:    "absent block uses safe default",
			content: "listen: \"127.0.0.1:8080\"\n",
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				require.Equal(t, ModeDefault, cfg.Permission.Mode)
				require.Empty(t, cfg.Permission.AllowedTools)
				require.Empty(t, cfg.Permission.DisallowedTools)
				require.Empty(t, cfg.Permission.AddDirs)
			},
		},
		{
			name:    "block parsed",
			content: "permission:\n  mode: acceptEdits\n  allowed_tools:\n    - Write\n    - \"Bash(git *)\"\n  add_dirs:\n    - /srv/work\n",
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				require.Equal(t, ModeAcceptEdits, cfg.Permission.Mode)
				require.Equal(t, []string{"Write", "Bash(git *)"}, cfg.Permission.AllowedTools)
				require.Equal(t, []string{"/srv/work"}, cfg.Permission.AddDirs)
			},
		},
		{
			name:    "empty mode normalized to default",
			content: "permission:\n  allowed_tools:\n    - Edit\n",
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				require.Equal(t, ModeDefault, cfg.Permission.Mode)
			},
		},
		{
			name:    "mcp tool spec accepted",
			content: "permission:\n  allowed_tools:\n    - mcp__github__create_issue\n",
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				require.Equal(t, []string{"mcp__github__create_issue"}, cfg.Permission.AllowedTools)
			},
		},
		{
			name:    "surrounding whitespace trimmed",
			content: "permission:\n  allowed_tools:\n    - \"  Write  \"\n",
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				require.Equal(t, []string{"Write"}, cfg.Permission.AllowedTools)
			},
		},
		{
			name:    "invalid mode rejected",
			content: "permission:\n  mode: yolo\n",
			wantErr: true,
		},
		{
			name:    "malformed tool spec rejected (unclosed rule)",
			content: "permission:\n  allowed_tools:\n    - \"Bash(\"\n",
			wantErr: true,
		},
		{
			name:    "malformed tool spec rejected (leading digit)",
			content: "permission:\n  allowed_tools:\n    - 123tool\n",
			wantErr: true,
		},
		{
			name:    "blank tool entry rejected",
			content: "permission:\n  disallowed_tools:\n    - \"   \"\n",
			wantErr: true,
		},
		{
			name:    "flag-like tool entry rejected",
			content: "permission:\n  allowed_tools:\n    - \"--dangerously-skip-permissions\"\n",
			wantErr: true,
		},
		{
			name:    "blank add_dirs entry rejected",
			content: "permission:\n  add_dirs:\n    - \"  \"\n",
			wantErr: true,
		},
		{
			name:    "flag-like add_dirs entry rejected",
			content: "permission:\n  add_dirs:\n    - \"-rf\"\n",
			wantErr: true,
		},
	}

	// Each supported mode must be accepted; add one table row per mode so they
	// run as named subtests alongside the rest.
	cases := make([]testCase, 0, len(static)+len(modes))
	cases = append(cases, static...)

	for _, mode := range modes {
		wantMode := mode
		cases = append(cases, testCase{
			name:    "supported mode " + mode + " accepted",
			content: "permission:\n  mode: " + mode + "\n",
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				require.Equal(t, wantMode, cfg.Permission.Mode)
			},
		})
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := writeFile(t, dir, "config.yaml", tc.content)

			cfg, err := Load(path)
			if tc.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)

			if tc.check != nil {
				tc.check(t, cfg)
			}
		})
	}
}
