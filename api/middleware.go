package api

import (
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/config"
	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/logr"
)

var (
	requestIdHeader string
)

// RequestLogConfig holds configuration for request logging
type RequestLogConfig struct {
	Enabled          bool
	RequestIDHeader  string
	LogUser          bool
	LogRequestBody   bool
	LogResponseBody  bool
	SensitiveHeaders []string
	ExcludedPaths    []string
}

// DefaultRequestLogConfig returns the default logging configuration
func DefaultRequestLogConfig() *RequestLogConfig {
	return &RequestLogConfig{
		Enabled:          true,
		RequestIDHeader:  config.Get("api.request-id.header", "X-Request-Id").String(),
		LogUser:          true,
		LogRequestBody:   false,
		LogResponseBody:  false,
		SensitiveHeaders: []string{"Authorization", "Cookie", "Set-Cookie"},
		ExcludedPaths:    []string{"/health", "/ok"},
	}
}

// LoggingMiddleware creates a framework-agnostic logging middleware
func LoggingMiddleware(log *logr.Logger, config *RequestLogConfig) MiddlewareFunc {
	if config == nil {
		config = DefaultRequestLogConfig()
	}

	return func(ctx contracts.RequestContext) (bool, error) {
		if !config.Enabled {
			return true, nil
		}

		start := time.Now()

		// Generate or extract request ID
		requestID := getOrCreateRequestID(ctx, config.RequestIDHeader)

		// Store request ID in context
		if ctxSetter, ok := ctx.(interface{ Set(string, interface{}) }); ok {
			ctxSetter.Set("request_id", requestID)
		}

		// Continue with the request
		if nexter, ok := ctx.(interface{ Next() }); ok {
			nexter.Next()
		}

		// Log after request completion
		logRequest(ctx, log, config, requestID, start)

		return true, nil
	}
}

// getOrCreateRequestID extracts or creates a request ID
func getOrCreateRequestID(ctx contracts.RequestContext, headerName string) string {
	// Try to get from header first
	if headerValue := ctx.Header(headerName); headerValue != "" {
		return headerValue
	}

	// Try common fallback headers
	commonHeaders := []string{"X-Request-Id", "Request-Id", "X-Request-ID"}
	for _, header := range commonHeaders {
		if headerValue := ctx.Header(header); headerValue != "" {
			return headerValue
		}
	}

	// Generate new UUID
	return uuid.New().String()
}

// logRequest logs the request details
func logRequest(ctx contracts.RequestContext, log *logr.Logger, config *RequestLogConfig, requestID string, start time.Time) {
	// Extract basic request information
	args := []interface{}{
		"duration", time.Since(start).Milliseconds(),
		"request_id", requestID,
	}

	// Add method and path if available
	if method := getRequestMethod(ctx); method != "" {
		args = append(args, "method", method)
	}
	if path := getRequestPath(ctx); path != "" {
		args = append(args, "path", path)
	}

	// Add status code if available
	if status := getResponseStatus(ctx); status != 0 {
		args = append(args, "status", status)
	}

	// Add client IP if available
	if clientIP := getClientIP(ctx); clientIP != "" {
		args = append(args, "client_ip", clientIP)
	}

	// Add user information if enabled and available
	if config.LogUser {
		if userID := getUserID(ctx); userID != "" {
			args = append(args, "user_id", userID)
		}
	}

	// Add response size if available
	if size := getResponseSize(ctx); size >= 0 {
		args = append(args, "size", size)
	}

	// Create log entry
	entry := log.With(args...)

	// Determine log level based on status code
	status := getResponseStatus(ctx)
	switch {
	case status >= 500:
		entry.Error("Server error")
	case status >= 400:
		entry.Warn("Client error")
	case shouldDebugLog(ctx, config):
		entry.Debug("Request processed")
	default:
		entry.Info("Request processed")
	}
}

// Helper functions to extract information from RequestContext
func getRequestMethod(ctx contracts.RequestContext) string {
	if reqCtx, ok := ctx.(interface{ Method() string }); ok {
		return reqCtx.Method()
	}
	return ""
}

func getRequestPath(ctx contracts.RequestContext) string {
	if reqCtx, ok := ctx.(interface{ Path() string }); ok {
		return reqCtx.Path()
	}
	return ""
}

func getResponseStatus(ctx contracts.RequestContext) int {
	if reqCtx, ok := ctx.(interface{ Status() int }); ok {
		return reqCtx.Status()
	}
	return 0
}

func getClientIP(ctx contracts.RequestContext) string {
	if reqCtx, ok := ctx.(interface{ ClientIP() string }); ok {
		return reqCtx.ClientIP()
	}
	return ""
}

func getResponseSize(ctx contracts.RequestContext) int64 {
	if reqCtx, ok := ctx.(interface{ Size() int64 }); ok {
		return reqCtx.Size()
	}
	return -1
}

func getUserID(ctx contracts.RequestContext) string {
	if ctxValue, ok := ctx.(interface{ Value(string) interface{} }); ok {
		if user := ctxValue.Value("auth.User"); user != nil {
			// We can't import auth here due to potential cycles, so we just return the string representation
			if userObj, ok := user.(interface{ ID() string }); ok {
				return userObj.ID()
			}
		}
	}
	return ""
}

func shouldDebugLog(ctx contracts.RequestContext, config *RequestLogConfig) bool {
	path := getRequestPath(ctx)
	return slices.Contains(config.ExcludedPaths, path)
}

// Legacy gin-compatible middleware for backward compatibility
func JSONLogMiddleware(log *logr.Logger) any {
	logConfig := DefaultRequestLogConfig()
	return LoggingMiddleware(log, logConfig)
}
