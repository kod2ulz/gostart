package logr

import (
	"context"
	"log/slog"
	"os"

	"github.com/google/uuid"
)

var (
	log         *Logger
	auditLogger *AuditLogger
)

type Logger struct {
	*slog.Logger
	host      string
	processID string
	traceID   string
}

type AuditLogger struct {
	writer AuditWriter
}

func (l *AuditLogger) Log(entry map[string]interface{}) {
	if l.writer != nil {
		_ = l.writer.Write(entry)
	}
}

func Log() *Logger {
	return log
}

func Audit() *AuditLogger {
	return auditLogger
}

func SetAuditWriter(w AuditWriter) {
	if auditLogger != nil {
		// Ensure the writer is non-blocking.
		auditEnv := logrEnv.Extend("AUDIT_LOG")
		auditLogger.writer = NewNonBlockingAuditWriter(w, auditEnv.Get("ASYNC_WRITER_BUFFER_MAX_SIZE", 1000).Int()) // Default buffer size of 1000
	}
}

func SetUpLogger(l *slog.Logger, w AuditWriter) {
	host, _ := os.Hostname()

	log = &Logger{
		Logger: l.With("host", host),
		host:   host,
	}
	auditLogger = &AuditLogger{}
	if w != nil {
		SetAuditWriter(w)
	}
}

// TID add trace_id field to log output
func (l *Logger) TID() *Logger {
	if l.traceID == "" {
		l.traceID = uuid.New().String()
	}
	l.Logger = l.With("trace_id", l.traceID)
	return l
}

// GetTID get trace_id field from logger
func (l *Logger) GetTID() string {
	return l.traceID
}

// WithTID set trace_id field to log output
func (l *Logger) WithTID(tid string) *Logger {
	l.Logger = l.With("trace_id", tid)
	return l
}

// ExtendWithTID returns a new logger with the trace_id field
func (l *Logger) ExtendWithTID(tid string) *Logger {
	return &Logger{
		Logger: l.Logger.With("trace_id", tid),
		host:   l.host,
	}
}

// ExtendWithField returns a new logger with the given field
func (l *Logger) ExtendWithField(field string, value interface{}) *Logger {
	return &Logger{
		Logger: l.Logger.With(field, value),
		host:   l.host,
	}
}

// PID add process_id field to log output
func (l *Logger) PID() *Logger {
	if l.processID == "" {
		l.processID = uuid.New().String()
	}
	l.Logger = l.With("process_id", l.processID)
	return l
}

// Ctx returns a new logger with context values
func (l *Logger) Ctx(ctx context.Context) *Logger {
	// This is a placeholder for context-aware logging if needed in the future
	return l
}
