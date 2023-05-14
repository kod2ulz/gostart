package logr 

import (
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const version = 1
const processIdField = "process_id"
const traceIdField = "trace_id"

var log *Logger

type Logger struct {
	*logrus.Entry
	host      string
	processID string
	traceID   string
}

func loggerCopy() *Logger {
	e := *log.Entry
	return &Logger{
		Entry: &e,
		host:  log.host,
	}
}

func Log() *Logger {
	return loggerCopy()
}

func SetUpLogger(l *logrus.Entry) error {
	host, level := _getHost(), _getLogLevel()

	log = &Logger{
		Entry: l,
		host:  host,
	}

	log.Entry = l.WithField("host", host)
	log.Logger.SetReportCaller(true)
	log.Logger.SetLevel(level)

	return nil
}

// SetFormatterJSON set JSON output format
func SetFormatterJSON() {
	log.Logger.SetFormatter(&logrus.JSONFormatter{})
}

// TID add trace_id field to log output
func (l *Logger) TID() *Logger {
	if l.traceID == "" {
		l.traceID = uuid.New().String()
	}
	l.Entry = l.WithField(traceIdField, l.traceID)
	return l
}

// GetTID get trace_id field from logger
func (l *Logger) GetTID() string {
	return l.traceID
}

// TID generate trace_id field to log output
func TID() *Logger {
	Log := loggerCopy()
	if Log.traceID == "" {
		Log.traceID = uuid.New().String()
	}
	Log.Entry = Log.WithField(traceIdField, Log.traceID)
	return Log
}

// WithTID set trace_id field to log output
func WithTID(tid string) *Logger {
	Log := loggerCopy()
	Log.Entry = Log.WithField(traceIdField, tid)
	return Log
}

// WithTID set trace_id field to log output
func (l *Logger) WithTID(tid string) *Logger {
	l.Entry = l.WithField(traceIdField, tid)
	return l
}

// ExtendWithTID set trace_id field to log output
func (l *Logger) ExtendWithTID(tid string) *Logger {
	e := *l.Entry
	out := &Logger{
		Entry: &e,
		host:  log.host,
	}
	out.Entry = l.WithField(traceIdField, tid)
	return out
}

// ExtendWithField set trace_id field to log output
func (l *Logger) ExtendWithField(field, value string) *Logger {
	e := *l.Entry
	out := &Logger{
		Entry: &e,
		host:  log.host,
	}
	out.Entry = l.WithField(field, value)
	return out
}

// PID add process_id field to log output
func (l *Logger) PID() *Logger {
	if l.processID == "" {
		l.processID = uuid.New().String()
	}
	l.Entry = l.WithField(processIdField, l.processID)
	return l
}

// PID generate process_id field to log output
func PID() *Logger {
	Log := loggerCopy()
	if Log.processID == "" {
		Log.processID = uuid.New().String()
	}
	Log.Entry = Log.WithField(processIdField, Log.processID)
	return Log
}

// InReqURL add incoming request url to log output
func (l *Logger) InReqURL(incomeRequestURL string) *Logger {
	l.Entry = l.WithField("income_request_url", incomeRequestURL)
	return l
}

// InReqURL add incoming request url to log output
func InReqURL(incomeRequestURL string) *Logger {
	Log := loggerCopy()
	Log.Entry = Log.WithField("income_request_url", incomeRequestURL)
	return Log
}

// OutReqURL add outcoming request url to log output
func (l *Logger) OutReqURL(outcomeRequestURL string) *Logger {
	l.Entry = l.WithField("outcome_request_url", outcomeRequestURL)
	return l
}

// OutReqURL add outcoming request url to log output
func OutReqURL(outcomeRequestURL string) *Logger {
	Log := loggerCopy()
	Log.Entry = Log.WithField("outcome_request_url", outcomeRequestURL)
	return Log
}

// FMsg full log message to log output
func (l Logger) FMsg(fullMsg string) Logger {
	l.Entry = l.WithField("full_message", fullMsg)
	return l
}

// FMsg full log message to log output
func FMsg(fullMsg string) Logger {
	Log := loggerCopy()
	Log.Entry = Log.WithField("full_message", fullMsg)
	return *Log
}


