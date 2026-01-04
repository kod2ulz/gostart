package gin

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/api/openapi"
	"github.com/kod2ulz/gostart/config"
	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/logr"
)

// GinRouter implements the api.Router interface using Gin
type GinRouter struct {
	engine          *gin.Engine
	group           *gin.RouterGroup
	openAPIRegistry *openapi.RouteRegistry
	openAPIConfig   *openapi.Info
	pathPrefix      string // Tracks the current path prefix from groups
}

// RequestContext implements both contracts.RequestContext and api.RequestContext
type RequestContext struct {
	*GinRequestContext
}

// NewGinRouter creates a new Gin-based router
func NewGinRouter(config *api.RouterConfig) (api.Router, error) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	// Initialize OpenAPI registry
	openAPIRegistry := openapi.NewRouteRegistry()

	// Default OpenAPI configuration
	openAPIConfig := &openapi.Info{
		Title:       "GoStart API",
		Description: "API generated automatically by GoStart",
		Version:     "1.0.0",
	}

	// Apply default middleware
	if config.EnableRecovery {
		engine.Use(gin.Recovery())
	}

	// Apply automatic logging middleware if enabled
	if config.EnableLogging {
		logConfig := config.LogConfig
		if logConfig == nil {
			logConfig = api.DefaultRequestLogConfig()
		}

		// Check config for logapi flag (defaulting to true)
		if shouldEnableLogging() {
			logger := logr.Log()
			// Skip logging if logger is not initialized (test environment)
			if logger != nil {
				loggingMiddleware := api.LoggingMiddleware(logger, logConfig)

				engine.Use(func(c *gin.Context) {
					ctx := &RequestContext{GinRequestContext: NewRequestContext(c).(*GinRequestContext)}
					if cont, err := loggingMiddleware(ctx); !cont || err != nil {
						if err != nil {
							c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]interface{}{
								"error": err.Error(),
							})
						}
						c.Abort()
						return
					}
					c.Next()
				})
			}
		}
	}

	// Apply custom middleware
	for _, mw := range config.CustomMiddleware {
		engine.Use(func(c *gin.Context) {
			ctx := &RequestContext{GinRequestContext: NewRequestContext(c).(*GinRequestContext)}
			if cont, err := mw(ctx); !cont || err != nil {
				if err != nil {
					c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]interface{}{
						"error": err.Error(),
					})
				}
				c.Abort()
				return
			}
			c.Next()
		})
	}

	router := &GinRouter{
		engine:          engine,
		openAPIRegistry: openAPIRegistry,
		openAPIConfig:   openAPIConfig,
	}

	// Configure static paths
	for path, root := range config.StaticPaths {
		router.engine.Static(path, root)
	}

	// Add OpenAPI documentation routes
	router.addOpenAPIRoutes()

	return router, nil
}

// HTTP Methods
func (r *GinRouter) GET(path string, handler api.HandlerFunc, options ...api.RouteOption) api.Router {
	return r.registerRoute("GET", path, handler, options...)
}

func (r *GinRouter) POST(path string, handler api.HandlerFunc, options ...api.RouteOption) api.Router {
	return r.registerRoute("POST", path, handler, options...)
}

func (r *GinRouter) PUT(path string, handler api.HandlerFunc, options ...api.RouteOption) api.Router {
	return r.registerRoute("PUT", path, handler, options...)
}

func (r *GinRouter) DELETE(path string, handler api.HandlerFunc, options ...api.RouteOption) api.Router {
	return r.registerRoute("DELETE", path, handler, options...)
}

func (r *GinRouter) PATCH(path string, handler api.HandlerFunc, options ...api.RouteOption) api.Router {
	return r.registerRoute("PATCH", path, handler, options...)
}

func (r *GinRouter) OPTIONS(path string, handler api.HandlerFunc, options ...api.RouteOption) api.Router {
	return r.registerRoute("OPTIONS", path, handler, options...)
}

func (r *GinRouter) HEAD(path string, handler api.HandlerFunc, options ...api.RouteOption) api.Router {
	return r.registerRoute("HEAD", path, handler, options...)
}

// Grouping
func (r *GinRouter) Group(path string, fn func(api.Router)) api.Router {
	// Create group from the current group, not from engine (to support nesting)
	var group *gin.RouterGroup
	if r.group != nil {
		group = r.group.Group(path)
	} else {
		group = r.engine.Group(path)
	}

	// Build the new path prefix
	newPrefix := r.pathPrefix
	if newPrefix == "" {
		newPrefix = "/" + path
	} else {
		newPrefix = newPrefix + "/" + path
	}
	// Ensure path starts with /
	if !strings.HasPrefix(newPrefix, "/") {
		newPrefix = "/" + newPrefix
	}

	subRouter := &GinRouter{
		engine:          r.engine,
		group:           group,
		openAPIRegistry: r.openAPIRegistry,
		openAPIConfig:   r.openAPIConfig,
		pathPrefix:      newPrefix,
	}
	fn(subRouter)
	return r
}

// Middleware
func (r *GinRouter) Use(middleware ...api.MiddlewareFunc) api.Router {
	for _, mw := range middleware {
		r.currentGroup().Use(func(c *gin.Context) {
			ctx := &RequestContext{GinRequestContext: NewRequestContext(c).(*GinRequestContext)}
			if cont, err := mw(ctx); !cont || err != nil {
				if err != nil {
					c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]interface{}{
						"error": err.Error(),
					})
				}
				c.Abort()
				return
			}
			c.Next()
		})
	}
	return r
}

// Static files
func (r *GinRouter) StaticFile(path, filePath string) api.Router {
	r.engine.StaticFile(path, filePath)
	return r
}

func (r *GinRouter) Static(prefix, root string) api.Router {
	r.engine.Static(prefix, root)
	return r
}

// Raw access
func (r *GinRouter) Underlying() any {
	return r.engine
}

// Run server
func (r *GinRouter) Run(addr string) error {
	return r.engine.Run(addr)
}

// Helper methods
func (r *GinRouter) registerRoute(method, path string, handler api.HandlerFunc, options ...api.RouteOption) api.Router {
	fullPath := r.buildFullPath(path)

	// Process options
	var (
		middlewares []api.MiddlewareFunc
		annotation  openapi.Annotation
		handlers    []api.HandlerFunc
	)

	for _, opt := range options {
		switch v := opt.(type) {
		case api.RouteMiddleware:
			middlewares = append(middlewares, v.Middleware)
		case api.RouteAnnotation:
			annotation = v.Annotation
		case api.RouteHandler:
			handlers = append(handlers, v.Handler)
		}
	}

	// Extract handler name for default summary
	handlerName := extractHandlerName(handler)
	if annotation.Summary == "" {
		annotation.Summary = handlerName
	}

	// Auto-generate documentation from typed handler using reflection
	annotation = r.enhanceAnnotationFromHandler(handler, annotation)

	// Merge with defaults (tags, etc.)
	annotation = r.mergeAnnotation(annotation, method, path)

	// Register with OpenAPI registry
	r.openAPIRegistry.Register(method, fullPath, handler, annotation)

	// Build the final handler chain
	finalHandler := handler
	for i := len(handlers) - 1; i >= 0; i-- {
		// Wrap handlers in reverse order so they execute in the right order
		// Use immediate function invocation to avoid closure capture issues
		currentHandler := handlers[i]
		nextHandler := finalHandler

		// Create a new closure with properly captured variables
		wrapped := func(ctx contracts.RequestContext) {
			currentHandler(ctx)
			// Note: We can't easily check if context was aborted
			// Handlers should call ctx.Abort() if they want to stop the chain
			nextHandler(ctx)
		}
		finalHandler = wrapped
	}

	// Wrap with middleware
	wrappedHandler := r.wrapHandlerWithMiddleware(finalHandler, middlewares)

	// Register the route with Gin
	switch method {
	case "GET":
		r.currentGroup().GET(path, wrappedHandler)
	case "POST":
		r.currentGroup().POST(path, wrappedHandler)
	case "PUT":
		r.currentGroup().PUT(path, wrappedHandler)
	case "DELETE":
		r.currentGroup().DELETE(path, wrappedHandler)
	case "PATCH":
		r.currentGroup().PATCH(path, wrappedHandler)
	case "OPTIONS":
		r.currentGroup().OPTIONS(path, wrappedHandler)
	case "HEAD":
		r.currentGroup().HEAD(path, wrappedHandler)
	}

	return r
}

func (r *GinRouter) currentGroup() *gin.RouterGroup {
	if r.group != nil {
		return r.group
	}
	return &r.engine.RouterGroup
}

func (r *GinRouter) wrapHandler(handler api.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := &RequestContext{GinRequestContext: NewRequestContext(c).(*GinRequestContext)}
		handler(ctx)
	}
}

// WrapHandler converts an api.HandlerFunc to gin.HandlerFunc for testing
// This is a convenience function for testing purposes
func WrapHandler(handler api.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := &RequestContext{GinRequestContext: NewRequestContext(c).(*GinRequestContext)}
		handler(ctx)
	}
}

// RequestContext implementations
func (ctx *RequestContext) Next() {
	ctx.ctx.Next()
}

func (ctx *RequestContext) Abort() {
	ctx.ctx.Abort()
}

func (ctx *RequestContext) AbortWithStatus(code int) {
	ctx.ctx.AbortWithStatus(code)
}

func (ctx *RequestContext) AbortWithStatusJSON(code int, obj any) {
	ctx.ctx.AbortWithStatusJSON(code, obj)
}

func (ctx *RequestContext) JSON(code int, obj any) {
	ctx.ctx.JSON(code, obj)
}

func (ctx *RequestContext) HTML(code int, name string, obj any) {
	ctx.ctx.HTML(code, name, obj)
}

func (ctx *RequestContext) String(code int, format string, values ...any) {
	ctx.ctx.String(code, format, values...)
}

func (ctx *RequestContext) Data(code int, contentType string, data []byte) {
	ctx.ctx.Data(code, contentType, data)
}

func (ctx *RequestContext) File(filepath string) {
	ctx.ctx.File(filepath)
}

func (ctx *RequestContext) SetHeader(key, value string) {
	ctx.ctx.Header(key, value)
}

func (ctx *RequestContext) Status(code int) {
	ctx.ctx.Status(code)
}

func (ctx *RequestContext) GetHeader(key string) string {
	return ctx.ctx.GetHeader(key)
}

func (ctx *RequestContext) SetCookie(cookie *http.Cookie) {
	http.SetCookie(ctx.ctx.Writer, cookie)
}

func (ctx *RequestContext) Cookie(name string) (string, error) {
	return ctx.ctx.Cookie(name)
}

func (ctx *RequestContext) ClientIP() string {
	return ctx.ctx.ClientIP()
}

// OpenAPIRouter interface implementation
func (r *GinRouter) GetOpenAPIHandler() http.Handler {
	return r.openAPIRegistry.GetOpenAPIJSONHandler("/openapi.json")
}

func (r *GinRouter) GetSwaggerUIHandler() http.Handler {
	return r.openAPIRegistry.GetSwaggerUIHandler("/swagger")
}

func (r *GinRouter) GenerateOpenAPIDoc() (*openapi.Document, error) {
	servers := []openapi.Server{
		{
			URL:         "http://localhost:8080",
			Description: "Development server",
		},
	}
	return r.openAPIRegistry.GenerateOpenAPIDoc(*r.openAPIConfig, servers)
}

func (r *GinRouter) SetOpenAPIInfo(info openapi.Info) {
	r.openAPIConfig = &info
}

// generateDefaultSummary generates a default summary for a route
func (r *GinRouter) generateDefaultSummary(method, path string) string {
	// Simple summary generation
	return fmt.Sprintf("%s %s", method, path)
}

// mergeAnnotation merges user-provided annotation with auto-generated defaults
func (r *GinRouter) mergeAnnotation(annotation openapi.Annotation, method, path string) openapi.Annotation {
	fullPath := r.buildFullPath(path)
	defaultTags := r.generateTagsFromPath(fullPath)

	// Start with the user's annotation
	merged := annotation

	// If no summary provided, use default
	if merged.Summary == "" {
		merged.Summary = r.generateDefaultSummary(method, path)
	}

	// If no tags provided, use auto-generated tags
	if len(merged.Tags) == 0 {
		merged.Tags = defaultTags
	}

	return merged
}

// buildFullPath builds the full path by combining the prefix with the given path
func (r *GinRouter) buildFullPath(path string) string {
	if r.pathPrefix == "" {
		return "/" + path
	}
	// Remove leading slash from path if present, since prefix already has it
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	return r.pathPrefix + "/" + path
}

// generateTagsFromPath generates tags from the path segments
func (r *GinRouter) generateTagsFromPath(path string) []string {
	// Remove leading slash and split
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")

	// Filter out empty strings and path parameters (starting with :)
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" && !strings.HasPrefix(part, ":") {
			tags = append(tags, part)
		}
	}

	// If no tags, use "default"
	if len(tags) == 0 {
		return []string{"default"}
	}

	return tags
}

// addOpenAPIRoutes adds routes for serving OpenAPI documentation
func (r *GinRouter) addOpenAPIRoutes() {
	// OpenAPI JSON endpoint
	r.engine.GET("/openapi.json", gin.WrapH(r.GetOpenAPIHandler()))

	// Swagger UI endpoint
	r.engine.GET("/swagger", gin.WrapH(r.GetSwaggerUIHandler()))

	// Redirect /swagger/ to /swagger
	r.engine.GET("/swagger/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger")
	})
}

// shouldEnableLogging checks if logging should be enabled based on config
func shouldEnableLogging() bool {
	// Default to true as requested by the user
	if value := config.Get("logapi"); value.Valid() {
		if strVal := value.String(); strVal == "false" || strVal == "0" {
			return false
		}
	}
	return true
}

// extractHandlerName extracts a clean handler name from the function
func extractHandlerName(handler api.HandlerFunc) string {
	// Get the function pointer and its name
	pc := reflect.ValueOf(handler).Pointer()
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "handler"
	}

	// Clean up the name
	name := fn.Name()

	// Remove package path
	parts := strings.Split(name, ".")
	if len(parts) > 0 {
		name = parts[len(parts)-1]
	}

	// Remove common suffixes like "-fm" (anonymous functions)
	name = regexp.MustCompile(`-fm\d*$`).ReplaceAllString(name, "")

	// Convert camelCase to Title Case for display
	name = regexp.MustCompile(`([a-z])([A-Z])`).ReplaceAllString(name, "$1 $2")
	name = strings.Title(name)

	return name
}

// enhanceAnnotationFromHandler uses reflection to auto-generate documentation from typed handlers
func (r *GinRouter) enhanceAnnotationFromHandler(handler api.HandlerFunc, annotation openapi.Annotation) openapi.Annotation {
	// Try to extract type information from the handler
	_ = reflect.ValueOf(handler)

	// Check if this is a typed handler (has generic type info embedded)
	// This is a placeholder for future enhancement where we inspect the handler's
	// parameter and response types using reflection
	//
	// For now, we'll rely on the OpenAPI registry which already does this inspection
	// when registering routes with typed handlers

	return annotation
}

// wrapHandlerWithMiddleware wraps a handler with route-specific middleware
func (r *GinRouter) wrapHandlerWithMiddleware(handler api.HandlerFunc, middlewares []api.MiddlewareFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := &RequestContext{GinRequestContext: NewRequestContext(c).(*GinRequestContext)}

		// Execute middleware in order
		for _, mw := range middlewares {
			cont, err := mw(ctx)
			if !cont || err != nil {
				if err != nil {
					c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]interface{}{
						"error": err.Error(),
					})
				}
				c.Abort()
				return
			}
		}

		// Execute the main handler
		handler(ctx)
	}
}

// Factory function for creating Gin routers
var NewRouter = NewGinRouter
