package logr

import (
	"encoding/json"
	"os"
	"sync"
)

// --- File Audit Writer ---

// FileAuditWriter is a simple AuditWriter that writes audit entries as JSON lines to a file.
type FileAuditWriter struct {
	file *os.File
	mu   sync.Mutex
}

// NewFileAuditWriter creates a new writer that appends audit logs to the specified file.
func NewFileAuditWriter(path string) (AuditWriter, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileAuditWriter{file: file},
		nil
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

// --- Non-Blocking Wrapper ---

// NonBlockingAuditWriter makes any AuditWriter asynchronous.
type NonBlockingAuditWriter struct {
	buffer chan map[string]interface{}
	child  AuditWriter
	wg     sync.WaitGroup
}

func NewNonBlockingAuditWriter(child AuditWriter, bufferSize int) *NonBlockingAuditWriter {
	w := &NonBlockingAuditWriter{
		buffer: make(chan map[string]interface{}, bufferSize),
		child:  child,
	}
	w.wg.Add(1)
	go w.process()
	return w
}

func (w *NonBlockingAuditWriter) process() {
	defer w.wg.Done()
	for entry := range w.buffer {
		_ = w.child.Write(entry)
	}
}

func (w *NonBlockingAuditWriter) Write(entry map[string]interface{}) error {
	select {
	case w.buffer <- entry:
		// Sent successfully
	default:
		// Buffer is full, message is dropped.
		// We could add a metric or internal log here if needed.
	}
	return nil
}

func (w *NonBlockingAuditWriter) Close() error {
	close(w.buffer)
	w.wg.Wait() // Wait for the processor to finish writing remaining entries.
	if closer, ok := w.child.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}
