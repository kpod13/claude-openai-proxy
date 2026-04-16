package logger_test

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/timur/claude-code-openai-server/internal/logger"
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

// TestInfoSuppressesDebug verifies that at INFO level, Debug messages are dropped.
func TestInfoSuppressesDebug(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	log := newLoggerWithBuf(&buf, false, false)
	log.Debug("should not appear")

	if buf.Len() != 0 {
		t.Errorf("expected no output at INFO level, got: %q", buf.String())
	}
}

// TestVerboseEmitsDebug verifies that at DEBUG level, Debug messages are emitted.
func TestVerboseEmitsDebug(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	log := newLoggerWithBuf(&buf, true, false)
	log.Debug("debug message")

	if !strings.Contains(buf.String(), "debug message") {
		t.Errorf("expected debug output, got: %q", buf.String())
	}
}

// TestQuietSuppressesAll verifies that quiet mode silences Info and Error.
func TestQuietSuppressesAll(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	log := newLoggerWithBuf(&buf, false, true)
	log.Info("info message")
	log.Error("error message")

	if buf.Len() != 0 {
		t.Errorf("expected no output in quiet mode, got: %q", buf.String())
	}
}

// TestQuietOverridesVerbose verifies quiet wins when both quiet and verbose are set.
func TestQuietOverridesVerbose(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	log := newLoggerWithBuf(&buf, true, true)
	log.Debug("debug message")
	log.Info("info message")

	if buf.Len() != 0 {
		t.Errorf("expected no output when quiet overrides verbose, got: %q", buf.String())
	}
}

// TestNew_PlainWritesToStderr verifies logger.New returns a working logger in plain mode.
func TestNew_PlainWritesToStderr(t *testing.T) {
	read := captureStderr(t)

	log := logger.New(false, false, "plain")
	log.Info("hello from plain")

	out := read()
	require.Contains(t, out, "hello from plain")
}

// TestNew_JSONWritesToStderr verifies logger.New returns a working logger in json mode.
func TestNew_JSONWritesToStderr(t *testing.T) {
	read := captureStderr(t)

	log := logger.New(false, false, "json")
	log.Info("hello from json")

	out := read()
	require.Contains(t, out, `"msg"`)
}

// TestNew_QuietProducesNoOutput verifies logger.New with quiet=true discards all output.
func TestNew_QuietProducesNoOutput(t *testing.T) {
	read := captureStderr(t)

	log := logger.New(false, true, "plain")
	log.Info("should be silent")
	log.Error("also silent")

	out := read()
	require.Empty(t, out)
}

// TestNew_VerboseEmitsDebug verifies logger.New with verbose=true emits debug messages.
func TestNew_VerboseEmitsDebug(t *testing.T) {
	read := captureStderr(t)

	log := logger.New(true, false, "plain")
	log.Debug("verbose debug message")

	out := read()
	require.Contains(t, out, "verbose debug message")
}
