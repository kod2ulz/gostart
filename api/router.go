package api

import (
	"net/http"

	"github.com/kod2ulz/gostart/contracts"
)

// Router defines a framework-agnostic router interface
type Router interface {
	// HTTP Methods
	GET(path string, handler HandlerFunc) Router
	POST(path string, handler HandlerFunc) Router
	PUT(path string, handler HandlerFunc) Router
	DELETE(path string, handler HandlerFunc) Router
	PATCH(path string, handler HandlerFunc) Router
	OPTIONS(path string, handler HandlerFunc) Router
	HEAD(path string, handler HandlerFunc) Router

	// Grouping
	Group(path string, fn func(Router)) Router

	// Middleware
	Use(middleware ...MiddlewareFunc) Router

	// Static files
	StaticFile(path, filePath string) Router
	Static(prefix, root string) Router

	// Raw access to underlying router
	Underlying() any

	// Run the server
	Run(addr string) error
}

// HandlerFunc represents a framework-agnostic handler function
type HandlerFunc func(contracts.RequestContext)

// MiddlewareFunc represents a framework-agnostic middleware function
type MiddlewareFunc func(contracts.RequestContext) (bool, error)

// RequestContext extends the contracts.RequestContext with router-specific methods
type RequestContext interface {
	contracts.RequestContext

	// Router-specific methods
	Next()
	Abort()
	AbortWithStatus(code int)
	AbortWithStatusJSON(code int, obj any)
	JSON(code int, obj any)
	HTML(code int, name string, obj any)
	String(code int, format string, values ...any)
	Data(code int, contentType string, data []byte)
	File(filepath string)
	SetHeader(key, value string)
	Status(code int)
	GetHeader(key string) string
	SetCookie(cookie *http.Cookie)
	Cookie(name string) (string, error)
	ClientIP() string
}

// RouterConfig holds router configuration
type RouterConfig struct {
	// CORS settings
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int

	// Middleware settings
	EnableRecovery   bool
	EnableLogging    bool
	LogConfig        *RequestLogConfig
	CustomMiddleware []MiddlewareFunc

	// Static file settings
	StaticPaths map[string]string
}

// RouterFactory creates a new router instance
type RouterFactory func(config *RouterConfig) (Router, error)