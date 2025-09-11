
package storage

import (
	"context"

	"github.com/jackc/pgx/v5/tracelog"
	"github.com/kod2ulz/gostart/logr"
	"github.com/sirupsen/logrus"
)

// pgxLoggerAdapter implements the pgx/v5 tracelog.Logger interface.
// It adapts pgx log messages to our structured logr.Logger.
type pgxLoggerAdapter struct {
	log *logr.Logger
}

// Log translates a pgx log call to a logr log call.
func (p *pgxLoggerAdapter) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]interface{}) {
	var logrusLevel logrus.Level
	switch level {
	case tracelog.LogLevelTrace, tracelog.LogLevelDebug:
		logrusLevel = logrus.DebugLevel
	case tracelog.LogLevelInfo:
		logrusLevel = logrus.InfoLevel
	case tracelog.LogLevelWarn:
		logrusLevel = logrus.WarnLevel
	case tracelog.LogLevelError:
		logrusLevel = logrus.ErrorLevel
	default:
		logrusLevel = logrus.InfoLevel // Default to Info
	}

	// Do not log trace-level messages unless our logger is configured for debug
	if level == tracelog.LogLevelTrace && p.log.Logger.GetLevel() < logrus.DebugLevel {
		return
	}

	p.log.WithFields(logrus.Fields(data)).Log(logrusLevel, msg)
}

// NewPgxLogger creates a tracer for pgx/v5 that is compatible with the application's logr logger.
func NewPgxLogger(log *logr.Logger) *tracelog.TraceLog {
	return &tracelog.TraceLog{
		Logger:   &pgxLoggerAdapter{log: log},
		LogLevel: tracelog.LogLevelInfo, // You can make this configurable
	}
}
