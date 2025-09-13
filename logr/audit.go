package logr

import (
	"encoding/json"
	"os"
	"sync"
)

// FileAuditWriter is a simple AuditWriter that writes audit entries as JSON lines to a file.
type FileAuditWriter struct {
	file *os.File
	mu   sync.Mutex
}

// NewFileAuditWriter creates a new writer that appends audit logs to the specified file.
// It creates the file if it doesn't exist.
func NewFileAuditWriter(path string) (AuditWriter, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileAuditWriter{file: file}, nil
}

// Write marshals the entry to JSON and writes it to the file, followed by a newline.
func (w *FileAuditWriter) Write(entry map[string]interface{}) error {
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	_, err = w.file.Write(append(line, '\n'))
	return err
}

// Close closes the underlying file.
func (w *FileAuditWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}
