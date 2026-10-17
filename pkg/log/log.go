// Package log constructs the application's slog logger.
package log

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
)

// New constructs the application logger writing to w.
//
// By default it emits stylized, colored output via the Tint handler, suitable
// for a terminal. Set MINECART_LOG_FORMAT=json to emit structured JSON instead
// (e.g. in containers). The level is controlled by MINECART_LOG_LEVEL
// (debug, info, warn, error) and defaults to debug.
func New(w io.Writer) *slog.Logger {
	lvl := level()

	if strings.EqualFold(os.Getenv("MINECART_LOG_FORMAT"), "json") {
		return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lvl}))
	}

	return slog.New(tint.NewTextHandler(w, &tint.Options{
		Level:      lvl,
		TimeFormat: time.RFC3339,
		AddSource:  true,
	}))
}

// level returns the configured log level, defaulting to debug.
func level() slog.Level {
	switch strings.ToLower(os.Getenv("MINECART_LOG_LEVEL")) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelDebug
	}
}
