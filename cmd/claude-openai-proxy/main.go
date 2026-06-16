package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/kpod13/claude-openai-proxy/internal/autorun"
	"github.com/kpod13/claude-openai-proxy/internal/config"
	"github.com/kpod13/claude-openai-proxy/internal/logger"
	"github.com/kpod13/claude-openai-proxy/internal/proxy"
	"github.com/kpod13/claude-openai-proxy/internal/ratelimit"
)

var (
	version = "dev"

	errNoModels      = errors.New("no Claude models discovered — ensure the claude CLI is on PATH and authenticated")
	errUnsupportedSh = errors.New("unsupported shell: choose from bash, zsh, fish, powershell")
)

type serverDeps struct {
	discover func(aliases []string) *proxy.Registry
	serve    func(srv *http.Server) error
}

func defaultDeps() serverDeps {
	return serverDeps{
		discover: proxy.Discover,
		serve:    func(srv *http.Server) error { return srv.ListenAndServe() },
	}
}

func newRootCmd(stdout io.Writer) *cobra.Command {
	return newRootCmdWith(stdout, defaultDeps())
}

func newRootCmdWith(stdout io.Writer, deps serverDeps) *cobra.Command {
	var (
		configPath string
		verbose    bool
		quiet      bool
		logFormat  string
		log        *slog.Logger
	)

	rootCmd := &cobra.Command{
		Use:     "claude-openai-proxy",
		Short:   "OpenAI-compatible proxy for Claude models",
		Version: version,
		Long: `claude-openai-proxy starts an HTTP server that exposes an OpenAI-compatible
API backed by Claude models via the claude CLI.

It translates /v1/chat/completions and /v1/models requests into Claude
subprocess calls, allowing OpenAI-compatible clients to use Claude.`,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			log = logger.New(verbose, quiet, logFormat)

			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			claudeVer, err := proxy.Version(context.Background())
			if err != nil {
				log.Warn("Could not determine Claude CLI version", "err", err)
			} else {
				log.Info("Claude CLI", "version", claudeVer)
			}

			log.Info("Discovering Claude models...")

			reg := deps.discover(cfg.Aliases)
			if reg.Len() == 0 {
				return errNoModels
			}

			modelIDs := make([]string, 0, reg.Len())
			for _, m := range reg.List() {
				modelIDs = append(modelIDs, m.ID)
			}

			log.Info("Models discovered", "count", reg.Len(), "models", modelIDs)

			h := &proxy.Handler{Registry: reg}

			if verbose {
				h.RunBlocking = proxy.DebugRunBlocking(log, proxy.RunBlocking)
				h.RunStreaming = proxy.DebugRunStreaming(log, proxy.RunStreaming)
				h.RunBlockingImages = proxy.DebugRunBlocking(log, proxy.RunBlockingImages)
				h.RunStreamingImages = proxy.DebugRunStreaming(log, proxy.RunStreamingImages)
			}

			limiter := ratelimit.New(
				cfg.RateLimit.RequestsPerMinute,
				cfg.RateLimit.TokensPerMinute,
			)
			rlMiddleware := ratelimit.Middleware(limiter)

			mux := http.NewServeMux()
			mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
				_, err := fmt.Fprint(w, "ok")
				if err != nil {
					http.Error(w, "write error", http.StatusInternalServerError)

					return
				}
			})
			mux.HandleFunc("/v1/models", h.Models)
			mux.Handle("/v1/chat/completions", rlMiddleware(http.HandlerFunc(h.ChatCompletions)))

			var handler http.Handler = mux
			if verbose {
				handler = proxy.DebugMiddleware(log)(mux)
			}

			srv := &http.Server{
				Addr:         cfg.Listen,
				Handler:      handler,
				ReadTimeout:  5 * time.Minute,
				WriteTimeout: 10 * time.Minute,
				IdleTimeout:  2 * time.Minute,
			}

			log.Info("Starting server", "addr", cfg.Listen)

			return runServer(cmd.Context(), log, srv, deps.serve)
		},
	}

	rootCmd.SetOut(stdout)

	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable debug-level log output")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "suppress all log output")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "plain", "log output format: plain or json")
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
				return rootCmd.GenBashCompletion(stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(stdout)
			case "fish":
				return rootCmd.GenFishCompletion(stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(stdout)
			default:
				return fmt.Errorf("%w: %q", errUnsupportedSh, args[0])
			}
		},
	}

	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(autorun.NewCmd(stdout))

	return rootCmd
}

const (
	// shutdownTimeout bounds how long graceful shutdown waits for in-flight
	// requests to drain before the process exits.
	shutdownTimeout = 30 * time.Second
)

// runServer starts srv via serve and shuts it down gracefully on SIGINT/SIGTERM
// or when the parent context is cancelled.
func runServer(ctx context.Context, log *slog.Logger, srv *http.Server, serve func(*http.Server) error) error {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() { serveErr <- serve(srv) }()

	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server error: %w", err)
		}

		return nil
	case <-ctx.Done():
		log.Info("Shutting down...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		err := srv.Shutdown(shutdownCtx)
		if err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}

		return nil
	}
}

func main() {
	cmd := newRootCmd(os.Stdout)

	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
