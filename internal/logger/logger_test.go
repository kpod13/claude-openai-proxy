package logger_test

import (
	"bytes"
	"io"
	"log/slog"
	"strings"
	"testing"
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
