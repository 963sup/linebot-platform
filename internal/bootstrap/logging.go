package bootstrap

import (
	"io"
	"log/slog"
)

// NewLogger creates a bounded-configuration logger. The caller owns the
// writer; runtime fields and credentials are never serialized here.
func NewLogger(cfg LogConfig, writer io.Writer) *slog.Logger {
	if writer == nil {
		writer = io.Discard
	}
	level := slog.LevelInfo
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	opts := &slog.HandlerOptions{Level: level}
	if cfg.Format == "json" {
		return slog.New(slog.NewJSONHandler(writer, opts))
	}
	return slog.New(slog.NewTextHandler(writer, opts))
}
