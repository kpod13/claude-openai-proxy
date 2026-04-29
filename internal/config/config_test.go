package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)

	err := os.WriteFile(path, []byte(content), 0o600)
	require.NoError(t, err)

	return path
}

func TestLoad(t *testing.T) {
	t.Parallel()

	t.Run("no file uses defaults", func(t *testing.T) {
		t.Parallel()

		cfg, err := Load("")
		require.NoError(t, err)
		require.Equal(t, "127.0.0.1:8080", cfg.Listen)
		require.Equal(t, []string{"opus", "sonnet", "haiku"}, cfg.Aliases)
		require.Equal(t, 0, cfg.RateLimit.RequestsPerMinute)
		require.Equal(t, 0, cfg.RateLimit.TokensPerMinute)
	})

	t.Run("missing explicit path returns error", func(t *testing.T) {
		t.Parallel()

		_, err := Load("/nonexistent/path/config.yaml")
		require.Error(t, err)
	})

	t.Run("explicit file", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name    string
			content string
			wantErr bool
			check   func(t *testing.T, cfg *Config)
		}{
			{
				name:    "valid YAML overrides defaults",
				content: "listen: \"0.0.0.0:9090\"\naliases:\n  - sonnet\n",
				check: func(t *testing.T, cfg *Config) {
					t.Helper()
					require.Equal(t, "0.0.0.0:9090", cfg.Listen)
					require.Equal(t, []string{"sonnet"}, cfg.Aliases)
				},
			},
			{
				name:    "partial YAML merges with defaults",
				content: `listen: "0.0.0.0:9090"`,
				check: func(t *testing.T, cfg *Config) {
					t.Helper()
					require.Equal(t, "0.0.0.0:9090", cfg.Listen)
					require.Equal(t, []string{"opus", "sonnet", "haiku"}, cfg.Aliases)
				},
			},
			{
				name:    "invalid YAML returns error",
				content: "listen: [invalid yaml",
				wantErr: true,
			},
			{
				name:    "rate limit parsed",
				content: "rate_limit:\n  requests_per_minute: 60\n  tokens_per_minute: 10000\n",
				check: func(t *testing.T, cfg *Config) {
					t.Helper()
					require.Equal(t, 60, cfg.RateLimit.RequestsPerMinute)
					require.Equal(t, 10000, cfg.RateLimit.TokensPerMinute)
				},
			},
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
	})
}

func TestLoad_SearchPath(t *testing.T) {
	cases := []struct {
		name    string
		content string
		wantErr bool
		check   func(t *testing.T, cfg *Config)
	}{
		{
			name:    "valid file",
			content: `listen: "0.0.0.0:7070"`,
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				require.Equal(t, "0.0.0.0:7070", cfg.Listen)
			},
		},
		{
			name:    "invalid YAML returns error",
			content: "listen: [bad",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeFile(t, dir, "search.yaml", tc.content)

			orig := searchPathsFn
			searchPathsFn = func() []string { return []string{path} }

			t.Cleanup(func() { searchPathsFn = orig })

			cfg, err := Load("")
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
