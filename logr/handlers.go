package logr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// LogstashHandler sends logs to Logstash via HTTP
type LogstashHandler struct {
	url        string
	httpClient *http.Client
	minLevel   slog.Level
}

// NewLogstashHandler creates a new Logstash handler
func NewLogstashHandler(url string, minLevel slog.Level) *LogstashHandler {
	return &LogstashHandler{
		url:        url,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		minLevel:   minLevel,
	}
}

// Enabled returns whether this handler is enabled for the given level
func (h *LogstashHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.minLevel
}

// Handle sends a log record to Logstash
func (h *LogstashHandler) Handle(ctx context.Context, record slog.Record) error {
	logEntry := make(map[string]interface{})
	logEntry["@timestamp"] = record.Time.Format(time.RFC3339)
	logEntry["level"] = record.Level.String()
	logEntry["message"] = record.Message
	logEntry["source"] = record.Source()

	// Add attributes
	record.Attrs(func(attr slog.Attr) bool {
		logEntry[attr.Key] = attr.Value.Any()
		return true
	})

	data, err := json.Marshal(logEntry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", h.url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		// Don't fail on network errors, just log to stderr
		fmt.Fprintf(os.Stderr, "failed to send log to Logstash: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		fmt.Fprintf(os.Stderr, "Logstash returned error status: %d\n", resp.StatusCode)
	}

	return nil
}

// WithAttrs returns a new handler with the given attributes
func (h *LogstashHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Create a new handler with the same configuration
	return &LogstashHandler{
		url:        h.url,
		httpClient: h.httpClient,
		minLevel:   h.minLevel,
	}
}

// WithGroup returns a new handler with the given group name
func (h *LogstashHandler) WithGroup(name string) slog.Handler {
	return h
}

// LokiHandler sends logs to Grafana Loki
type LokiHandler struct {
	url        string
	httpClient *http.Client
	minLevel   slog.Level
	labels     map[string]string
}

// NewLokiHandler creates a new Loki handler
func NewLokiHandler(url string, minLevel slog.Level, labels map[string]string) *LokiHandler {
	if labels == nil {
		labels = make(map[string]string)
	}
	// Add default labels if not present
	if _, ok := labels["app"]; !ok {
		labels["app"] = "gostart"
	}
	if _, ok := labels["host"]; !ok {
		hostname, _ := os.Hostname()
		labels["host"] = hostname
	}

	return &LokiHandler{
		url:        url,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		minLevel:   minLevel,
		labels:     labels,
	}
}

// Enabled returns whether this handler is enabled for the given level
func (h *LokiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.minLevel
}

// Handle sends a log record to Loki
func (h *LokiHandler) Handle(ctx context.Context, record slog.Record) error {
	// Build log entry
	logEntry := make(map[string]interface{})
	logEntry["level"] = record.Level.String()
	logEntry["message"] = record.Message
	if record.PC != 0 {
		logEntry["source"] = record.Source()
	}

	// Add attributes
	record.Attrs(func(attr slog.Attr) bool {
		logEntry[attr.Key] = attr.Value.Any()
		return true
	})

	logLine, err := json.Marshal(logEntry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	// Build Loki push request
	lokiReq := map[string]interface{}{
		"streams": []map[string]interface{}{
			{
				"stream": h.labels,
				"values": [][]string{
					{
						fmt.Sprintf("%d", record.Time.UnixNano()),
						string(logLine),
					},
				},
			},
		},
	}

	data, err := json.Marshal(lokiReq)
	if err != nil {
		return fmt.Errorf("failed to marshal Loki request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", h.url+"/loki/api/v1/push", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to send log to Loki: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		fmt.Fprintf(os.Stderr, "Loki returned error status: %d\n", resp.StatusCode)
	}

	return nil
}

// WithAttrs returns a new handler with the given attributes
func (h *LokiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LokiHandler{
		url:        h.url,
		httpClient: h.httpClient,
		minLevel:   h.minLevel,
		labels:     h.labels,
	}
}

// WithGroup returns a new handler with the given group name
func (h *LokiHandler) WithGroup(name string) slog.Handler {
	return h
}

// SentryHandler sends error logs to Sentry
type SentryHandler struct {
	dsn        string
	httpClient *http.Client
	minLevel   slog.Level
	environment string
	release     string
}

// NewSentryHandler creates a new Sentry handler
func NewSentryHandler(dsn string, minLevel slog.Level, environment, release string) *SentryHandler {
	return &SentryHandler{
		dsn:         dsn,
		httpClient:  &http.Client{Timeout: 5 * time.Second},
		minLevel:    minLevel,
		environment: environment,
		release:     release,
	}
}

// Enabled returns whether this handler is enabled for the given level
func (h *SentryHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.minLevel
}

// Handle sends a log record to Sentry
func (h *SentryHandler) Handle(ctx context.Context, record slog.Record) error {
	// Only send errors and above to Sentry
	if record.Level < slog.LevelError {
		return nil
	}

	// Build Sentry event
	event := map[string]interface{}{
		"event_id":    generateEventID(),
		"timestamp":   record.Time.Unix(),
		"level":       sentryLevel(record.Level),
		"message":     record.Message,
		"environment": h.environment,
		"release":     h.release,
		"tags":        make(map[string]interface{}),
		"extra":       make(map[string]interface{}),
	}

	// Add attributes as tags or extra data
	record.Attrs(func(attr slog.Attr) bool {
		if shouldBeTag(attr.Key) {
			tags := event["tags"].(map[string]interface{})
			tags[attr.Key] = attr.Value.Any()
		} else {
			extra := event["extra"].(map[string]interface{})
			extra[attr.Key] = attr.Value.Any()
		}
		return true
	})

	// Add source information
	if record.PC != 0 {
		source := record.Source()
		event["extra"].(map[string]interface{})["source"] = fmt.Sprintf("%s:%d", source.File, source.Line)
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal Sentry event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", h.dsn, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to send event to Sentry: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		fmt.Fprintf(os.Stderr, "Sentry returned error status: %d\n", resp.StatusCode)
	}

	return nil
}

// WithAttrs returns a new handler with the given attributes
func (h *SentryHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &SentryHandler{
		dsn:         h.dsn,
		httpClient:  h.httpClient,
		minLevel:    h.minLevel,
		environment: h.environment,
		release:     h.release,
	}
}

// WithGroup returns a new handler with the given group name
func (h *SentryHandler) WithGroup(name string) slog.Handler {
	return h
}

// Helper functions

func sentryLevel(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return "debug"
	case slog.LevelInfo:
		return "info"
	case slog.LevelWarn:
		return "warning"
	case slog.LevelError:
		return "error"
	default:
		if level >= slog.LevelError+4 {
			return "fatal"
		}
		return "error"
	}
}

func shouldBeTag(key string) bool {
	// Common keys that should be tags in Sentry
	tags := map[string]bool{
		"host":        true,
		"environment": true,
		"user_id":     true,
		"request_id":  true,
		"trace_id":    true,
		"service":     true,
	}
	return tags[key]
}

func generateEventID() string {
	return fmt.Sprintf("%032x", time.Now().UnixNano())
}

// BufferedHandler wraps another handler and buffers log entries for batch sending
type BufferedHandler struct {
	handler    slog.Handler
	buffer     []slog.Record
	maxSize    int
	flushTimer *time.Timer
	flushChan  chan struct{}
}

// NewBufferedHandler creates a new buffered handler
func NewBufferedHandler(handler slog.Handler, bufferSize int, flushInterval time.Duration) *BufferedHandler {
	h := &BufferedHandler{
		handler:   handler,
		buffer:    make([]slog.Record, 0, bufferSize),
		maxSize:   bufferSize,
		flushChan: make(chan struct{}, 1),
	}

	// Start flush timer
	h.flushTimer = time.AfterFunc(flushInterval, func() {
		h.flush()
		h.flushTimer.Reset(flushInterval)
	})

	return h
}

// Enabled returns whether this handler is enabled for the given level
func (h *BufferedHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// Handle buffers a log record
func (h *BufferedHandler) Handle(ctx context.Context, record slog.Record) error {
	h.buffer = append(h.buffer, record)

	if len(h.buffer) >= h.maxSize {
		h.flush()
	}

	return nil
}

// flush sends all buffered records to the underlying handler
func (h *BufferedHandler) flush() {
	if len(h.buffer) == 0 {
		return
	}

	ctx := context.Background()
	for _, record := range h.buffer {
		h.handler.Handle(ctx, record)
	}

	h.buffer = h.buffer[:0]
}

// WithAttrs returns a new handler with the given attributes
func (h *BufferedHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &BufferedHandler{
		handler:   h.handler.WithAttrs(attrs),
		buffer:    h.buffer,
		maxSize:   h.maxSize,
		flushChan: h.flushChan,
	}
}

// WithGroup returns a new handler with the given group name
func (h *BufferedHandler) WithGroup(name string) slog.Handler {
	return &BufferedHandler{
		handler:   h.handler.WithGroup(name),
		buffer:    h.buffer,
		maxSize:   h.maxSize,
		flushChan: h.flushChan,
	}
}

// Stop stops the buffered handler and flushes remaining entries
func (h *BufferedHandler) Stop() {
	h.flushTimer.Stop()
	h.flush()
}
