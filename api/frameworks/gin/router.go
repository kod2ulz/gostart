package gin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/api/openapi"
	"github.com/kod2ulz/gostart/config"
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
func (r *GinRouter) GET(path string, handler api.HandlerFunc) api.Router {
	// Build full path
	fullPath := r.buildFullPath(path)
	// Generate tags from path
	tags := r.generateTagsFromPath(fullPath)

	// Register with OpenAPI registry
	annotation := openapi.Annotation{
		Summary: r.generateDefaultSummary("GET", path),
		Tags:    tags,
	}
	r.openAPIRegistry.Register("GET", fullPath, handler, annotation)

	r.currentGroup().GET(path, r.wrapHandler(handler))
	return r
}

func (r *GinRouter) POST(path string, handler api.HandlerFunc) api.Router {
	// Build full path
	fullPath := r.buildFullPath(path)
	// Generate tags from path
	tags := r.generateTagsFromPath(fullPath)

	// Register with OpenAPI registry
	annotation := openapi.Annotation{
		Summary: r.generateDefaultSummary("POST", path),
		Tags:    tags,
	}
	r.openAPIRegistry.Register("POST", fullPath, handler, annotation)

	r.currentGroup().POST(path, r.wrapHandler(handler))
	return r
}

func (r *GinRouter) PUT(path string, handler api.HandlerFunc) api.Router {
	// Build full path
	fullPath := r.buildFullPath(path)
	// Generate tags from path
	tags := r.generateTagsFromPath(fullPath)

	// Register with OpenAPI registry
	annotation := openapi.Annotation{
		Summary: r.generateDefaultSummary("PUT", path),
		Tags:    tags,
	}
	r.openAPIRegistry.Register("PUT", fullPath, handler, annotation)

	r.currentGroup().PUT(path, r.wrapHandler(handler))
	return r
}

func (r *GinRouter) DELETE(path string, handler api.HandlerFunc) api.Router {
	// Build full path
	fullPath := r.buildFullPath(path)
	// Generate tags from path
	tags := r.generateTagsFromPath(fullPath)

	// Register with OpenAPI registry
	annotation := openapi.Annotation{
		Summary: r.generateDefaultSummary("DELETE", path),
		Tags:    tags,
	}
	r.openAPIRegistry.Register("DELETE", fullPath, handler, annotation)

	r.currentGroup().DELETE(path, r.wrapHandler(handler))
	return r
}

func (r *GinRouter) PATCH(path string, handler api.HandlerFunc) api.Router {
	// Build full path
	fullPath := r.buildFullPath(path)
	// Generate tags from path
	tags := r.generateTagsFromPath(fullPath)

	// Register with OpenAPI registry
	annotation := openapi.Annotation{
		Summary: r.generateDefaultSummary("PATCH", path),
		Tags:    tags,
	}
	r.openAPIRegistry.Register("PATCH", fullPath, handler, annotation)

	r.currentGroup().PATCH(path, r.wrapHandler(handler))
	return r
}

func (r *GinRouter) OPTIONS(path string, handler api.HandlerFunc) api.Router {
	// Build full path
	fullPath := r.buildFullPath(path)
	// Generate tags from path
	tags := r.generateTagsFromPath(fullPath)

	// Register with OpenAPI registry
	annotation := openapi.Annotation{
		Summary: r.generateDefaultSummary("OPTIONS", path),
		Tags:    tags,
	}
	r.openAPIRegistry.Register("OPTIONS", fullPath, handler, annotation)

	r.currentGroup().OPTIONS(path, r.wrapHandler(handler))
	return r
}

func (r *GinRouter) HEAD(path string, handler api.HandlerFunc) api.Router {
	// Build full path
	fullPath := r.buildFullPath(path)
	// Generate tags from path
	tags := r.generateTagsFromPath(fullPath)

	// Register with OpenAPI registry
	annotation := openapi.Annotation{
		Summary: r.generateDefaultSummary("HEAD", path),
		Tags:    tags,
	}
	r.openAPIRegistry.Register("HEAD", fullPath, handler, annotation)

	r.currentGroup().HEAD(path, r.wrapHandler(handler))
	return r
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

// generateDefaultSummary generates a default summary for a route
func (r *GinRouter) generateDefaultSummary(method, path string) string {
	// Simple summary generation
	return fmt.Sprintf("%s %s", method, path)
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

// Factory function for creating Gin routers
var NewRouter = NewGinRouter
