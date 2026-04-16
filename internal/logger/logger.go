package logger

import (
	"io"
	"log/slog"
	"os"
)

// New constructs a *slog.Logger with the requested level and format.
// quiet=true silences all output (takes precedence over verbose).
// verbose=true sets the level to DEBUG; otherwise INFO is used.
// format must be "json" for JSON output; anything else uses plain text.
func New(verbose, quiet bool, format string) *slog.Logger {
	var w io.Writer = os.Stderr
	if quiet {
		w = io.Discard
	}

	level := slog.LevelInfo
	if verbose && !quiet {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	return slog.New(handler)
}
