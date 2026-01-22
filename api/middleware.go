package api

import (
	"fmt"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/config"
	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/logr"
)

const (
	// Context key for storing the current handler name
	contextKeyHandler = "current_handler"
	// Context key for storing error location (file:line)
	contextKeyErrorLocation = "error_location"
)

// RequestLogConfig holds configuration for request logging
type RequestLogConfig struct {
	Enabled            bool
	RequestIDHeader    string   // Primary header to check for request ID
	RequestIDFallbacks []string // Fallback headers to check if primary not found (common standards)
	LogUser            bool
	LogRequestBody     bool
	LogResponseBody    bool
	LogHandler         bool   // Log which handler/function processed the request
	SensitiveHeaders   []string
	ExcludedPaths      []string
}

// DefaultRequestLogConfig returns the default logging configuration
func DefaultRequestLogConfig() *RequestLogConfig {
	// Get configured primary header
	primaryHeader := config.Get("api.request-id.header", "X-Request-Id").String()

	// Get fallback headers from config (comma-separated) or use defaults
	fallbackConfig := config.Get("api.request-id.fallbacks", "X-Request-Id,Request-Id,X-Request-ID").String()
	fallbacks := splitAndTrim(fallbackConfig, ",")

	return &RequestLogConfig{
		Enabled:            true,
		RequestIDHeader:    primaryHeader,
		RequestIDFallbacks: fallbacks,
		LogUser:            true,
		LogRequestBody:     false,
		LogResponseBody:    false,
		LogHandler:         true, // Enable handler logging by default
		SensitiveHeaders:   []string{"Authorization", "Cookie", "Set-Cookie"},
		ExcludedPaths:      []string{"/health", "/ok"},
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

		// Generate or extract request ID (uses config with fallbacks)
		requestID := getOrCreateRequestID(ctx, config)

		// Store request ID in context
		if ctxSetter, ok := ctx.(interface{ Set(string, any) }); ok {
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
func getOrCreateRequestID(ctx contracts.RequestContext, config *RequestLogConfig) string {
	// Try to get from primary header first
	if headerValue := ctx.Header(config.RequestIDHeader); headerValue != "" {
		return headerValue
	}

	// Try configured fallback headers (for upstream compatibility)
	for _, header := range config.RequestIDFallbacks {
		// Skip if it's the same as primary (avoid duplicate check)
		if header == config.RequestIDHeader {
			continue
		}
		if headerValue := ctx.Header(header); headerValue != "" {
			return headerValue
		}
	}

	// Generate new UUID
	return uuid.New().String()
}

// logRequest logs the request details
func logRequest(ctx contracts.RequestContext, log *logr.Logger, config *RequestLogConfig, requestID string, start time.Time) {
	duration := time.Since(start)

	// Duration in seconds as float64 for Grafana compatibility
	// 0.001 = 1ms, 1 = 1s
	durationSec := float64(duration.Nanoseconds()) / 1e9

	// Extract basic request information
	args := []any{
		"duration", durationSec,
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
	status := getResponseStatus(ctx)
	if status != 0 {
		args = append(args, "status", status)
	}

	// Add client IP if available
	if clientIP := getClientIP(ctx); clientIP != "" {
		args = append(args, "client_ip", clientIP)
	}

	// Add handler name if enabled
	if config.LogHandler {
		if handler := getHandlerName(ctx); handler != "" {
			args = append(args, "handler", handler)
		}
	}

	// Add error location (file:line) for error responses
	if status >= 400 {
		if errorLoc := getErrorLocation(ctx); errorLoc != "" {
			args = append(args, "error_at", errorLoc)
		}
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
	if ctxValue, ok := ctx.(interface{ Value(string) any }); ok {
		if user := ctxValue.Value("auth.User"); user != nil {
			// We can't import auth here due to potential cycles, so we just return the string representation
			if userObj, ok := user.(interface{ GetID() string }); ok {
				return userObj.GetID()
			}
			if userObj, ok := user.(interface{ GetID() uuid.UUID }); ok {
				return userObj.GetID().String()
			}
		}
	}
	return ""
}

func shouldDebugLog(ctx contracts.RequestContext, config *RequestLogConfig) bool {
	path := getRequestPath(ctx)
	return slices.Contains(config.ExcludedPaths, path)
}

// getHandlerName extracts the handler name from context
func getHandlerName(ctx contracts.RequestContext) string {
	if ctxValue, ok := ctx.(interface{ Value(any) any }); ok {
		if handler := ctxValue.Value(contextKeyHandler); handler != nil {
			return fmt.Sprintf("%v", handler)
		}
	}
	return ""
}

// getErrorLocation extracts the error location (file:line) from context
func getErrorLocation(ctx contracts.RequestContext) string {
	if ctxValue, ok := ctx.(interface{ Value(any) any }); ok {
		if loc := ctxValue.Value(contextKeyErrorLocation); loc != nil {
			return fmt.Sprintf("%v", loc)
		}
	}
	return ""
}

// SetErrorLocation stores the error location in context
// Should be called when an error occurs.
// Uses runtime.Callers to find the actual error origin (skips framework internals)
func SetErrorLocation(ctx contracts.RequestContext, file string, line int) {
	if ctxSetter, ok := ctx.(interface{ Set(string, any) }); ok {
		// If file and line are provided, use them
		if file != "" && line > 0 {
			location := fmt.Sprintf("%s:%d", file, line)
			ctxSetter.Set(contextKeyErrorLocation, location)
			return
		}

		// Otherwise, try to find the error origin automatically
		// Skip 2 frames to get past this function and its caller
		location := findErrorOrigin(2)
		if location != "" {
			ctxSetter.Set(contextKeyErrorLocation, location)
		}
	}
}

// findErrorLocation searches the call stack for the actual error origin
// Skips framework internals to find user code
func findErrorOrigin(skipFrames int) string {
	start := skipFrames + 1
	maxDepth := 15

	for depth := start; depth < start+maxDepth; depth++ {
		pc, file, line, ok := runtime.Caller(depth)
		if !ok {
			break
		}

		// Skip framework internals
		if isFrameworkFunction(runtime.FuncForPC(pc).Name()) {
			continue
		}

		// Found user code - return file:line
		if file != "" {
			return fmt.Sprintf("%s:%d", file, line)
		}
	}

	return ""
}

// SetHandlerName sets the current handler name in context
// This should be called by handler wrappers to track which function is processing the request
func SetHandlerName(ctx contracts.RequestContext, name string) {
	if ctxSetter, ok := ctx.(interface{ Set(string, any) }); ok {
		// Try to get a cleaner function name
		cleanName := getFunctionName(name)
		ctxSetter.Set(contextKeyHandler, cleanName)
	}
}

// getFunctionName extracts a clean function name from a full function path
// e.g., "github.com/yourapp/main.TypedHelloHandler" -> "main.TypedHelloHandler"
func getFunctionName(fullPath string) string {
	// Get just the function name without package path
	parts := strings.Split(fullPath, ".")
	if len(parts) == 0 {
		return fullPath
	}

	funcName := parts[len(parts)-1]

	// If we have at least 2 parts (package.function), return "package.function"
	if len(parts) >= 2 {
		packageName := parts[len(parts)-2]
		return fmt.Sprintf("%s.%s", packageName, funcName)
	}

	return funcName
}

// GetCallersHandlerName returns the name of the calling function (skip levels up the stack)
// This is a convenience function for handler wrappers to automatically track their name
// Uses a heuristic to find the actual user handler (skips framework internals)
func GetCallersHandlerName(skipFrames int) string {
	// Start searching from the caller's position
	start := skipFrames + 1
	maxDepth := 10 // Don't go too deep

	for depth := start; depth < start+maxDepth; depth++ {
		pc, _, _, ok := runtime.Caller(depth)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		fullName := fn.Name()

		// Skip framework internals and anonymous functions
		if isFrameworkFunction(fullName) {
			continue
		}

		// Found the user's handler function
		return getFunctionName(fullName)
	}

	return "unknown"
}

// isFrameworkFunction checks if a function name is from the gostart framework
func isFrameworkFunction(fullName string) bool {
	// Skip anonymous functions (e.g., .func1, .func2)
	if strings.Contains(fullName, ".func") {
		return true
	}

	// Skip known framework packages
	frameworkPaths := []string{
		"github.com/kod2ulz/gostart",
		// "net/http",
	}

	for _, prefix := range frameworkPaths {
		if strings.Contains(fullName, prefix) {
			return true
		}
	}

	return false
}

// Legacy gin-compatible middleware for backward compatibility
func JSONLogMiddleware(log *logr.Logger) any {
	logConfig := DefaultRequestLogConfig()
	return LoggingMiddleware(log, logConfig)
}

// splitAndTrim splits a string by a separator and trims each part
func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
