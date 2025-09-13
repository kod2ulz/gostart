package logr

import (
	"log/slog"
	"os"
	"strings"
)

// Config initializes the logger with default settings.
func Config() error {
	levelStr := os.Getenv("LOG_LEVEL")
	if levelStr == "" {
		levelStr = "info"
	}
	level := _getLogLevel(levelStr)

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
	})

	logger := slog.New(handler)

	// For now, the audit writer is nil. It will be configured separately.
	SetUpLogger(logger, nil)

	return nil
}

func _getLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func _getHost() (string, error) {
	return os.Hostname()
}

// These functions are kept for backward compatibility but are no longer used by the new slog-based configuration.

func _getLogrusLogLevel(level string) int {
	// Kept for reference, but not used.
	return 0
}

func _getLogrusHost() string {
	// Kept for reference, but not used.
	return ""
}