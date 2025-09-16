package gin

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/api/openapi"
	"github.com/kod2ulz/gostart/config"
	"github.com/kod2ulz/gostart/logr"
)

// GinRouter implements the api.Router interface using Gin
type GinRouter struct {
	engine         *gin.Engine
	group          *gin.RouterGroup
	openAPIRegistry *openapi.RouteRegistry
	openAPIConfig  *openapi.Info
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
		engine:         engine,
		openAPIRegistry: openAPIRegistry,
		openAPIConfig:  openAPIConfig,
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
	// Register with OpenAPI registry
	annotation := openapi.Annotation{
		Summary: r.generateDefaultSummary("GET", path),
		Tags:    []string{"default"},
	}
	r.openAPIRegistry.Register("GET", path, handler, annotation)

	r.currentGroup().GET(path, r.wrapHandler(handler))
	return r
}

func (r *GinRouter) POST(path string, handler api.HandlerFunc) api.Router {
	// Register with OpenAPI registry
	annotation := openapi.Annotation{
		Summary: r.generateDefaultSummary("POST", path),
		Tags:    []string{"default"},
	}
	r.openAPIRegistry.Register("POST", path, handler, annotation)

	r.currentGroup().POST(path, r.wrapHandler(handler))
	return r
}

func (r *GinRouter) PUT(path string, handler api.HandlerFunc) api.Router {
	// Register with OpenAPI registry
	annotation := openapi.Annotation{
		Summary: r.generateDefaultSummary("PUT", path),
		Tags:    []string{"default"},
	}
	r.openAPIRegistry.Register("PUT", path, handler, annotation)

	r.currentGroup().PUT(path, r.wrapHandler(handler))
	return r
}

func (r *GinRouter) DELETE(path string, handler api.HandlerFunc) api.Router {
	// Register with OpenAPI registry
	annotation := openapi.Annotation{
		Summary: r.generateDefaultSummary("DELETE", path),
		Tags:    []string{"default"},
	}
	r.openAPIRegistry.Register("DELETE", path, handler, annotation)

	r.currentGroup().DELETE(path, r.wrapHandler(handler))
	return r
}

func (r *GinRouter) PATCH(path string, handler api.HandlerFunc) api.Router {
	// Register with OpenAPI registry
	annotation := openapi.Annotation{
		Summary: r.generateDefaultSummary("PATCH", path),
		Tags:    []string{"default"},
	}
	r.openAPIRegistry.Register("PATCH", path, handler, annotation)

	r.currentGroup().PATCH(path, r.wrapHandler(handler))
	return r
}

func (r *GinRouter) OPTIONS(path string, handler api.HandlerFunc) api.Router {
	// Register with OpenAPI registry
	annotation := openapi.Annotation{
		Summary: r.generateDefaultSummary("OPTIONS", path),
		Tags:    []string{"default"},
	}
	r.openAPIRegistry.Register("OPTIONS", path, handler, annotation)

	r.currentGroup().OPTIONS(path, r.wrapHandler(handler))
	return r
}

func (r *GinRouter) HEAD(path string, handler api.HandlerFunc) api.Router {
	// Register with OpenAPI registry
	annotation := openapi.Annotation{
		Summary: r.generateDefaultSummary("HEAD", path),
		Tags:    []string{"default"},
	}
	r.openAPIRegistry.Register("HEAD", path, handler, annotation)

	r.currentGroup().HEAD(path, r.wrapHandler(handler))
	return r
}

// Grouping
func (r *GinRouter) Group(path string, fn func(api.Router)) api.Router {
	group := r.engine.Group(path)
	subRouter := &GinRouter{engine: r.engine, group: group}
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