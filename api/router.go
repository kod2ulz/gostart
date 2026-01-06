package api

import (
	"net/http"

	"github.com/kod2ulz/gostart/api/openapi"
	"github.com/kod2ulz/gostart/config"
	"github.com/kod2ulz/gostart/contracts"
)

// Router defines a framework-agnostic router interface
type Router interface {
	// HTTP Methods (with optional route configuration)
	GET(path string, handler HandlerFunc, options ...RouteOption) Router
	POST(path string, handler HandlerFunc, options ...RouteOption) Router
	PUT(path string, handler HandlerFunc, options ...RouteOption) Router
	DELETE(path string, handler HandlerFunc, options ...RouteOption) Router
	PATCH(path string, handler HandlerFunc, options ...RouteOption) Router
	OPTIONS(path string, handler HandlerFunc, options ...RouteOption) Router
	HEAD(path string, handler HandlerFunc, options ...RouteOption) Router

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

// RouteOption represents an optional route configuration (middleware, annotation, or additional handler)
type RouteOption interface {
	isRouteOption()
}

// RouteMiddleware wraps a MiddlewareFunc as a RouteOption
type RouteMiddleware struct {
	Middleware MiddlewareFunc
}

func (r RouteMiddleware) isRouteOption() {}

// RouteAnnotation wraps an openapi.Annotation as a RouteOption
type RouteAnnotation struct {
	Annotation openapi.Annotation
}

func (r RouteAnnotation) isRouteOption() {}

// RouteHandler wraps an additional HandlerFunc as a RouteOption (for chaining)
type RouteHandler struct {
	Handler HandlerFunc
}

func (r RouteHandler) isRouteOption() {}

// Helper functions to create RouteOptions
func WithMiddleware(mw MiddlewareFunc) RouteOption {
	return RouteMiddleware{Middleware: mw}
}

func WithAnnotation(annotation openapi.Annotation) RouteOption {
	return RouteAnnotation{Annotation: annotation}
}

func WithHandler(handler HandlerFunc) RouteOption {
	return RouteHandler{Handler: handler}
}

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
	NativeMiddleware []any // Native framework middleware (e.g., gin.Recovery(), cors.New())

	// Panic recovery settings
	PanicRecovery    func(c any, stack []byte) // Custom panic recovery handler
	// If PanicRecovery is set, EnableRecovery will be set to false to use custom handler instead
	// You can also pass native gin.Recovery() or gin.CustomRecovery() in NativeMiddleware

	// Static file settings
	StaticPaths map[string]string
}

// RouterFactory creates a new router instance
type RouterFactory func(config *RouterConfig) (Router, error)

// OpenAPIRouter extends the Router interface with OpenAPI documentation support
type OpenAPIRouter interface {
	Router
	// OpenAPI documentation methods
	GetOpenAPIHandler() http.Handler
	GetSwaggerUIHandler() http.Handler
	GenerateOpenAPIDoc() (*openapi.Document, error)
	SetOpenAPIInfo(info openapi.Info)
}

// DefaultRouterConfig creates a router configuration from environment variables
func DefaultRouterConfig(router_prefix...string) *RouterConfig {
	env := config.Env.Helper(router_prefix...).OrDefault("HTTP_SERVER")

	return &RouterConfig{
		// CORS settings from environment
		AllowOrigins:     env.Get("ALLOW_ORIGINS", "*").StringList(","),
		AllowMethods:     env.Get("ALLOW_METHODS", "GET,POST,PUT,HEAD,OPTIONS").StringList(","),
		AllowHeaders:     env.Get("ALLOW_HEADERS", "Origin,Content-Length,Accept-Encoding,Authorization,Accept-Language,Content-Type").StringList(","),
		ExposeHeaders:    env.Get("EXPOSE_HEADERS", "Content-Length,Host,Content-Type,Connection").StringList(","),
		AllowCredentials: env.Get("ALLOW_CREDENTIALS", "true").Bool(),
		MaxAge:           int(env.Get("MAX_AGE", "12h").Duration().Seconds()),

		// Middleware settings
		EnableRecovery:   env.Get("ENABLE_RECOVERY", "true").Bool(),
		EnableLogging:    env.Get("ENABLE_LOGGING", "true").Bool(),
		LogConfig:        DefaultRequestLogConfig(),
		CustomMiddleware: []MiddlewareFunc{},

		// Static file settings
		StaticPaths: make(map[string]string),
	}
}
