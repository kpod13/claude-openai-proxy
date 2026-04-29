package autorun

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteDefaultConfigIfAbsent(t *testing.T) {
	cases := []struct {
		name        string
		setup       func(t *testing.T) string
		wantErr     bool
		errContains string
		wantCreated bool
		check       func(t *testing.T, home string)
	}{
		{
			name: "stat error",
			setup: func(t *testing.T) string {
				t.Helper()
				home := fileAsHome(t)
				t.Setenv("HOME", home)

				return home
			},
			wantErr:     true,
			errContains: "stat config",
		},
		{
			name: "creates file",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				t.Setenv("HOME", dir)

				return dir
			},
			wantCreated: true,
			check: func(t *testing.T, home string) {
				t.Helper()

				content, err := os.ReadFile(filepath.Join(home, defaultConfigName))
				require.NoError(t, err)
				require.Contains(t, string(content), "listen:")
				require.Contains(t, string(content), "127.0.0.1:8080")
			},
		},
		{
			name: "skips existing file",
			setup: func(t *testing.T) string {
				t.Helper()

				dir := t.TempDir()
				t.Setenv("HOME", dir)

				original := []byte("listen: \"0.0.0.0:9090\"\n")
				err := os.WriteFile(filepath.Join(dir, defaultConfigName), original, 0o600)
				require.NoError(t, err)

				return dir
			},
			wantCreated: false,
			check: func(t *testing.T, home string) {
				t.Helper()

				content, err := os.ReadFile(filepath.Join(home, defaultConfigName))
				require.NoError(t, err)
				require.Equal(t, []byte("listen: \"0.0.0.0:9090\"\n"), content)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := tc.setup(t)

			created, err := WriteDefaultConfigIfAbsent()

			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantCreated, created)

			if tc.check != nil {
				tc.check(t, home)
			}
		})
	}
}
