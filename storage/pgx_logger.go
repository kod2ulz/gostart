package storage

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/tracelog"
	"github.com/kod2ulz/gostart/logr"
)

// pgxLoggerAdapter implements the pgx/v5 tracelog.Logger interface.
// It adapts pgx log messages to our structured logr.Logger.
type pgxLoggerAdapter struct {
	log *logr.Logger
}

// Log translates a pgx log call to a logr log call.
func (p *pgxLoggerAdapter) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]interface{}) {
	var slogLevel slog.Level
	switch level {
	case tracelog.LogLevelTrace, tracelog.LogLevelDebug:
		slogLevel = slog.LevelDebug
	case tracelog.LogLevelInfo:
		slogLevel = slog.LevelInfo
	case tracelog.LogLevelWarn:
		slogLevel = slog.LevelWarn
	case tracelog.LogLevelError:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo // Default to Info
	}

	if !p.log.Enabled(ctx, slogLevel) {
		return
	}

	args := make([]interface{}, 0, len(data)*2)
	for k, v := range data {
		args = append(args, k, v)
	}

	p.log.Log(ctx, slogLevel, msg, args...)
}

// NewPgxLogger creates a tracer for pgx/v5 that is compatible with the application's logr logger.
func NewPgxLogger(log *logr.Logger) *tracelog.TraceLog {
	return &tracelog.TraceLog{
		Logger:   &pgxLoggerAdapter{log: log},
		LogLevel: tracelog.LogLevelInfo, // You can make this configurable
	}
}
