package autorun

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/timur/claude-code-openai-server/internal/proxy"
)

// NewCmd builds the "autorun" Cobra command with "install" and "uninstall"
// subcommands.
func NewCmd(stdout io.Writer) *cobra.Command {
	autorunCmd := &cobra.Command{
		Use:   "autorun",
		Short: "Manage user-level autostart for the proxy",
		Long: `autorun manages the OS-specific user-level autostart entry for the proxy.

The entry runs the proxy automatically when the current user logs in.
No root or administrator privileges are required.`,
	}

	autorunCmd.AddCommand(newInstallCmd(stdout))
	autorunCmd.AddCommand(newUninstallCmd(stdout))

	return autorunCmd
}

func newInstallCmd(stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Register the proxy as a user-level autostart entry",
		RunE: func(cmd *cobra.Command, _ []string) error {
			binPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("autorun install: resolve binary path: %w", err)
			}

			backend, err := New()
			if err != nil {
				return fmt.Errorf("autorun install: %w", err)
			}

			cfg := InstallConfig{
				BinaryPath: binPath,
				Label:      "claude-openai-proxy",
			}

			err = backend.Install(cmd.Context(), cfg)
			if err != nil {
				return fmt.Errorf("autorun install: %w", err)
			}

			created, err := WriteDefaultConfigIfAbsent()
			if err != nil {
				return fmt.Errorf("autorun install: write config: %w", err)
			}

			claudeVer, err := proxy.Version(cmd.Context())
			if err != nil {
				_, err = fmt.Fprintf(stdout, "Warning: could not determine Claude CLI version: %v\n", err)
				if err != nil {
					return fmt.Errorf("autorun install: write output: %w", err)
				}
			} else {
				_, err = fmt.Fprintf(stdout, "Claude CLI version: %s\n", claudeVer)
				if err != nil {
					return fmt.Errorf("autorun install: write output: %w", err)
				}
			}

			_, err = fmt.Fprintf(stdout, "Autorun installed for %s\n", binPath)
			if err != nil {
				return fmt.Errorf("autorun install: write output: %w", err)
			}

			if created {
				_, err = fmt.Fprintf(stdout, "Default config written to ~/%s\n", defaultConfigName)
			} else {
				_, err = fmt.Fprintf(stdout, "Existing config at ~/%s was not modified\n", defaultConfigName)
			}

			if err != nil {
				return fmt.Errorf("autorun install: write output: %w", err)
			}

			return nil
		},
	}
}

func newUninstallCmd(stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the user-level autostart entry",
		RunE: func(cmd *cobra.Command, _ []string) error {
			backend, err := New()
			if err != nil {
				return fmt.Errorf("autorun uninstall: %w", err)
			}

			err = backend.Uninstall(cmd.Context())
			if err != nil {
				return fmt.Errorf("autorun uninstall: %w", err)
			}

			_, err = fmt.Fprintln(stdout, "Autorun uninstalled")
			if err != nil {
				return fmt.Errorf("autorun uninstall: write output: %w", err)
			}

			return nil
		},
	}
}
