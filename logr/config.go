package logr

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/kod2ulz/gostart/config"
	"github.com/lmittmann/tint"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logrEnv = config.Env.Helper()
)

// --- Multi Handler ---

// MultiHandler dispatches log records to multiple handlers.
type MultiHandler struct {
	handlers []slog.Handler
}

func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	return &MultiHandler{handlers: handlers}
}

func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *MultiHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, handler := range h.handlers {
		_ = handler.Handle(ctx, record)
	}
	return nil
}

func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}
	return NewMultiHandler(newHandlers...)
}

func (h *MultiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}
	return NewMultiHandler(newHandlers...)
}



// NewConsoleHandler creates a handler that writes to stdout.
// If pretty is true, it uses a colorized, human-friendly format.
// Otherwise, it writes minified JSON.
func NewConsoleHandler(pretty bool) slog.Handler {
	
	level := _getLogLevel(logrEnv.GetString("LOG_LEVEL", "info"))
	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
	}

	if pretty {
		return tint.NewHandler(os.Stdout, &tint.Options{
			AddSource:  true,
			Level:      level,
			TimeFormat: time.Kitchen,
		})
	}
	return slog.NewJSONHandler(os.Stdout, opts)
}

// NewFileHandler creates a handler that writes to a file with rotation.
// If rotation is nil, default settings are read from environment variables.
func NewFileHandler(path string, rotation *RotationConfig) slog.Handler {
	if rotation == nil {
		fileHandlerEnv := logrEnv.Extend("LOG_ROTATE")
		rotation = &RotationConfig{
			MaxSize:    fileHandlerEnv.Get("MAX_SIZE", 100).Int(),
			MaxBackups: fileHandlerEnv.Get("MAX_BACKUPS", 5).Int(),
			MaxAge:     fileHandlerEnv.Get("MAX_AGE", 30).Int(),
			Compress:   fileHandlerEnv.Get("COMPRESS", true).Bool(),
		}
	}

	writer := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    rotation.MaxSize,
		MaxAge:     rotation.MaxAge,
		MaxBackups: rotation.MaxBackups,
		Compress:   rotation.Compress,
	}

	return slog.NewJSONHandler(writer, &slog.HandlerOptions{
		AddSource: true,
		Level:     _getLogLevel(logrEnv.GetString("LOG_LEVEL", "info")),
	})
}

// --- Main Config ---

// Config initializes the logger.
// It can take multiple handlers, which will all receive log entries.
// If no handlers are provided, it defaults to a JSON handler to stdout.
func Config(handlers ...slog.Handler) error {
	var finalHandler slog.Handler

	if len(handlers) == 0 {
		finalHandler = NewConsoleHandler(false) // Default to minified JSON
	} else if len(handlers) == 1 {
		finalHandler = handlers[0]
	} else {
		finalHandler = NewMultiHandler(handlers...)
	}

	logger := slog.New(finalHandler)

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