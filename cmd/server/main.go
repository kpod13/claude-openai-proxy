package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/timur/claude-code-openai-server/internal/config"
	"github.com/timur/claude-code-openai-server/internal/proxy"
)

var (
	version = "dev"

	errNoModels      = errors.New("no Claude models discovered — ensure the claude CLI is on PATH and authenticated")
	errUnsupportedSh = errors.New("unsupported shell: choose from bash, zsh, fish, powershell")
)

func main() {
	var configPath string

	rootCmd := &cobra.Command{
		Use:     "claude-openai-proxy",
		Short:   "OpenAI-compatible proxy for Claude models",
		Version: version,
		Long: `claude-openai-proxy starts an HTTP server that exposes an OpenAI-compatible
API backed by Claude models via the claude CLI.

It translates /v1/chat/completions and /v1/models requests into Claude
subprocess calls, allowing OpenAI-compatible clients to use Claude.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			log.Println("Discovering Claude models...")

			reg := proxy.Discover(cfg.Aliases)
			if reg.Len() == 0 {
				return errNoModels
			}

			log.Printf("Discovered %d model(s)", reg.Len())

			h := &proxy.Handler{Registry: reg}

			mux := http.NewServeMux()
			mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
				_, err := fmt.Fprint(w, "ok")
				if err != nil {
					http.Error(w, "write error", http.StatusInternalServerError)

					return
				}
			})
			mux.HandleFunc("/v1/models", h.Models)
			mux.HandleFunc("/v1/chat/completions", h.ChatCompletions)

			srv := &http.Server{
				Addr:         cfg.Listen,
				Handler:      mux,
				ReadTimeout:  5 * time.Minute,
				WriteTimeout: 10 * time.Minute,
				IdleTimeout:  2 * time.Minute,
			}

			log.Printf("Starting server on %s", cfg.Listen)

			err = srv.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("server error: %w", err)
			}

			return nil
		},
	}

	rootCmd.Flags().StringVar(&configPath, "config", "", "path to config file (default: search standard locations)")

	completionCmd := &cobra.Command{
		Use:   "completion <shell>",
		Short: "Generate shell autocompletion script",
		Long: `Generate an autocompletion script for claude-openai-proxy for the specified shell.

Supported shells: bash, zsh, fish, powershell

Installation instructions:

  Bash:
    claude-openai-proxy completion bash > /etc/bash_completion.d/claude-openai-proxy
    # or for the current user:
    claude-openai-proxy completion bash > ~/.bash_completion

  Zsh:
    claude-openai-proxy completion zsh > "${fpath[1]}/_claude-openai-proxy"
    # or add to ~/.zshrc:
    source <(claude-openai-proxy completion zsh)

  Fish:
    claude-openai-proxy completion fish > ~/.config/fish/completions/claude-openai-proxy.fish

  PowerShell:
    claude-openai-proxy completion powershell | Out-String | Invoke-Expression
`,
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(_ *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("%w: %q", errUnsupportedSh, args[0])
			}
		},
	}

	rootCmd.AddCommand(completionCmd)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
