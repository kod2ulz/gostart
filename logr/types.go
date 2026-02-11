package logr

// AuditWriter defines the interface for writing audit logs to a destination.
type AuditWriter interface {
	Write(entry map[string]interface{}) error
}

// RotationConfig configures the log file rotation.
type RotationConfig struct {
	MaxSize    int  // Max size in megabytes before rotation
	MaxAge     int  // Max number of days to retain old log files
	MaxBackups int  // Max number of old log files to retain
	Compress   bool // Whether to compress/gzip old log files
}
