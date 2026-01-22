package logr

import (
	"context"
	"encoding/json"
	"fmt"
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

// NewPrettyJSONHandler creates a handler that writes formatted JSON to stdout
func NewPrettyJSONHandler(opts *slog.HandlerOptions) slog.Handler {
	return &prettyJSONHandlerWrapper{opts: opts}
}

// prettyJSONHandlerWrapper wraps the slog.Handler to produce pretty JSON output
type prettyJSONHandlerWrapper struct {
	opts *slog.HandlerOptions
}

func (h *prettyJSONHandlerWrapper) Enabled(ctx context.Context, level slog.Level) bool {
	if h.opts == nil || h.opts.Level == nil {
		return true
	}
	return level >= h.opts.Level.Level()
}

func (h *prettyJSONHandlerWrapper) Handle(ctx context.Context, record slog.Record) error {
	// Build a map from the record
	attrs := make(map[string]interface{})

	// Add time
	attrs["time"] = record.Time.Format(time.RFC3339)

	// Add level
	attrs["level"] = record.Level.String()

	// Add message
	attrs["msg"] = record.Message

	// Add source if requested
	if h.opts != nil && h.opts.AddSource {
		source := record.Source()
		attrs["source"] = map[string]string{
			"function": source.Function,
			"file":     source.File,
			"line":     fmt.Sprintf("%d", source.Line),
		}
	}

	// Add all attributes
	record.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})

	// Marshal to pretty JSON
	jsonBytes, err := json.MarshalIndent(attrs, "", "  ")
	if err != nil {
		return err
	}

	// Write to stdout with newline
	os.Stdout.Write(append(jsonBytes, '\n'))
	return nil
}

func (h *prettyJSONHandlerWrapper) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &prettyJSONHandlerWrapper{opts: h.opts}
}

func (h *prettyJSONHandlerWrapper) WithGroup(name string) slog.Handler {
	return &prettyJSONHandlerWrapper{opts: h.opts}
}

// NewConsoleHandler creates a handler that writes to stdout.
// Supported formats:
//   - "text" or "pretty": colored, human-friendly format (tint)
//   - "json": minified JSON
//   - "pretty-json" or "json-pretty": formatted/indented JSON
func NewConsoleHandler(format string) slog.Handler {

	level := _getLogLevel(logrEnv.GetString("LOG_LEVEL", "info"))
	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
	}

	switch strings.ToLower(format) {
	case "text", "pretty":
		return tint.NewHandler(os.Stdout, &tint.Options{
			AddSource:  true,
			Level:      level,
			TimeFormat: time.Kitchen,
		})
	case "json", "":
		return slog.NewJSONHandler(os.Stdout, opts)
	case "pretty-json", "json-pretty":
		// Pretty JSON with indentation
		return NewPrettyJSONHandler(opts)
	default:
		// Default to JSON if format not recognized
		return slog.NewJSONHandler(os.Stdout, opts)
	}
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
// If no handlers are provided, it uses the LOG_FORMAT environment variable.
// Supported LOG_FORMAT values:
//   - "json" (default): minified JSON
//   - "pretty-json": formatted/indented JSON
//   - "text" or "pretty": colored, human-friendly format
func Config(handlers ...slog.Handler) error {
	var finalHandler slog.Handler

	if len(handlers) == 0 {
		// Check LOG_FORMAT environment variable (default to "json")
		format := logrEnv.GetString("LOG_FORMAT", "json")
		finalHandler = NewConsoleHandler(format)
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
