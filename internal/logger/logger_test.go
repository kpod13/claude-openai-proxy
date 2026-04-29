package logger_test

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/kpod13/claude-openai-proxy/internal/logger"
	"github.com/stretchr/testify/require"
)

// newLoggerWithBuf builds a plain-text logger that writes to buf.
func newLoggerWithBuf(buf *bytes.Buffer, verbose, quiet bool) *slog.Logger {
	var w io.Writer = buf
	if quiet {
		w = io.Discard
	}

	level := slog.LevelInfo
	if verbose && !quiet {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{Level: level}

	return slog.New(slog.NewTextHandler(w, opts))
}

// captureStderr redirects os.Stderr to a pipe for the duration of the test
// and returns a function to read what was captured. Call the returned function
// exactly once to flush and read captured output.
func captureStderr(t *testing.T) func() string {
	t.Helper()

	r, w, err := os.Pipe()
	require.NoError(t, err)

	old := os.Stderr
	os.Stderr = w

	return func() string {
		os.Stderr = old

		require.NoError(t, w.Close())

		var buf bytes.Buffer

		_, _ = io.Copy(&buf, r)
		require.NoError(t, r.Close())

		return buf.String()
	}
}

func TestLoggerLevels(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		verbose bool
		quiet   bool
		log     func(l *slog.Logger)
		wantMsg string
	}{
		{
			name:    "INFO suppresses debug",
			verbose: false,
			quiet:   false,
			log:     func(l *slog.Logger) { l.Debug("should not appear") },
		},
		{
			name:    "verbose emits debug",
			verbose: true,
			quiet:   false,
			log:     func(l *slog.Logger) { l.Debug("debug message") },
			wantMsg: "debug message",
		},
		{
			name:    "quiet suppresses all",
			verbose: false,
			quiet:   true,
			log: func(l *slog.Logger) {
				l.Info("info message")
				l.Error("error message")
			},
		},
		{
			name:    "quiet overrides verbose",
			verbose: true,
			quiet:   true,
			log: func(l *slog.Logger) {
				l.Debug("debug message")
				l.Info("info message")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			l := newLoggerWithBuf(&buf, tc.verbose, tc.quiet)
			tc.log(l)

			if tc.wantMsg != "" {
				require.Contains(t, buf.String(), tc.wantMsg)
			} else {
				require.Empty(t, buf.String())
			}
		})
	}
}

func TestNew(t *testing.T) {
	cases := []struct {
		name    string
		verbose bool
		quiet   bool
		format  string
		log     func(l *slog.Logger)
		wantMsg string
	}{
		{
			name:    "plain writes to stderr",
			verbose: false,
			quiet:   false,
			format:  "plain",
			log:     func(l *slog.Logger) { l.Info("hello from plain") },
			wantMsg: "hello from plain",
		},
		{
			name:    "JSON writes to stderr",
			verbose: false,
			quiet:   false,
			format:  "json",
			log:     func(l *slog.Logger) { l.Info("hello from json") },
			wantMsg: `"msg"`,
		},
		{
			name:   "quiet produces no output",
			verbose: false,
			quiet:  true,
			format: "plain",
			log: func(l *slog.Logger) {
				l.Info("should be silent")
				l.Error("also silent")
			},
		},
		{
			name:    "verbose emits debug",
			verbose: true,
			quiet:   false,
			format:  "plain",
			log:     func(l *slog.Logger) { l.Debug("verbose debug message") },
			wantMsg: "verbose debug message",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			read := captureStderr(t)

			l := logger.New(tc.verbose, tc.quiet, tc.format)
			tc.log(l)

			out := read()

			if tc.wantMsg != "" {
				require.Contains(t, out, tc.wantMsg)
			} else {
				require.Empty(t, out)
			}
		})
	}
}
