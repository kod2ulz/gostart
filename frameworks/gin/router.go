package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kod2ulz/gostart/api/ginadapter"
	"github.com/kod2ulz/gostart/router"
)

// GinRouter implements the app.Router interface using Gin
type GinRouter struct {
	engine *gin.Engine
	group  *gin.RouterGroup
}

// GinRequestContext implements both contracts.RequestContext and app.RequestContext
type GinRequestContext struct {
	*ginadapter.GinRequestContext
}

// NewGinRouter creates a new Gin-based router
func NewGinRouter(config *router.RouterConfig) (router.Router, error) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	// Apply default middleware
	if config.EnableRecovery {
		engine.Use(gin.Recovery())
	}

// Apply custom middleware
	for _, mw := range config.CustomMiddleware {
		engine.Use(func(c *gin.Context) {
			ctx := &GinRequestContext{GinRequestContext: ginadapter.NewRequestContext(c).(*ginadapter.GinRequestContext)}
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

	router := &GinRouter{engine: engine}

	// Configure static paths
	for path, root := range config.StaticPaths {
		router.engine.Static(path, root)
	}

	return router, nil
}

// HTTP Methods
func (r *GinRouter) GET(path string, handler router.RouterHandlerFunc) router.Router {
	r.currentGroup().GET(path, r.wrapHandler(handler))
	return r
}

func (r *GinRouter) POST(path string, handler router.RouterHandlerFunc) router.Router {
	r.currentGroup().POST(path, r.wrapHandler(handler))
	return r
}

func (r *GinRouter) PUT(path string, handler router.RouterHandlerFunc) router.Router {
	r.currentGroup().PUT(path, r.wrapHandler(handler))
	return r
}

func (r *GinRouter) DELETE(path string, handler router.RouterHandlerFunc) router.Router {
	r.currentGroup().DELETE(path, r.wrapHandler(handler))
	return r
}

func (r *GinRouter) PATCH(path string, handler router.RouterHandlerFunc) router.Router {
	r.currentGroup().PATCH(path, r.wrapHandler(handler))
	return r
}

func (r *GinRouter) OPTIONS(path string, handler router.RouterHandlerFunc) router.Router {
	r.currentGroup().OPTIONS(path, r.wrapHandler(handler))
	return r
}

func (r *GinRouter) HEAD(path string, handler router.RouterHandlerFunc) router.Router {
	r.currentGroup().HEAD(path, r.wrapHandler(handler))
	return r
}

// Grouping
func (r *GinRouter) Group(path string, fn func(router.Router)) router.Router {
	group := r.engine.Group(path)
	subRouter := &GinRouter{engine: r.engine, group: group}
	fn(subRouter)
	return r
}

// Middleware
func (r *GinRouter) Use(middleware ...router.MiddlewareFunc) router.Router {
	for _, mw := range middleware {
		r.currentGroup().Use(func(c *gin.Context) {
			ctx := &GinRequestContext{GinRequestContext: ginadapter.NewRequestContext(c).(*ginadapter.GinRequestContext)}
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
func (r *GinRouter) StaticFile(path, filePath string) router.Router {
	r.engine.StaticFile(path, filePath)
	return r
}

func (r *GinRouter) Static(prefix, root string) router.Router {
	r.engine.Static(prefix, root)
	return r
}

// Raw access
func (r *GinRouter) Router() any {
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

func (r *GinRouter) wrapHandler(handler router.RouterHandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := &GinRequestContext{GinRequestContext: ginadapter.NewRequestContext(c).(*ginadapter.GinRequestContext)}
		handler(ctx)
	}
}

// RequestContext implementations
func (ctx *GinRequestContext) Next() {
	ctx.GinContext.Next()
}

func (ctx *GinRequestContext) Abort() {
	ctx.GinContext.Abort()
}

func (ctx *GinRequestContext) AbortWithStatus(code int) {
	ctx.GinContext.AbortWithStatus(code)
}

func (ctx *GinRequestContext) AbortWithStatusJSON(code int, obj interface{}) {
	ctx.GinContext.AbortWithStatusJSON(code, obj)
}

func (ctx *GinRequestContext) JSON(code int, obj interface{}) {
	ctx.GinContext.JSON(code, obj)
}

func (ctx *GinRequestContext) HTML(code int, name string, obj interface{}) {
	ctx.GinContext.HTML(code, name, obj)
}

func (ctx *GinRequestContext) String(code int, format string, values ...interface{}) {
	ctx.GinContext.String(code, format, values...)
}

func (ctx *GinRequestContext) Data(code int, contentType string, data []byte) {
	ctx.GinContext.Data(code, contentType, data)
}

func (ctx *GinRequestContext) File(filepath string) {
	ctx.GinContext.File(filepath)
}

func (ctx *GinRequestContext) SetHeader(key, value string) {
	ctx.GinContext.Header(key, value)
}

func (ctx *GinRequestContext) Status(code int) {
	ctx.GinContext.Status(code)
}

func (ctx *GinRequestContext) GetHeader(key string) string {
	return ctx.GinContext.GetHeader(key)
}

func (ctx *GinRequestContext) SetCookie(cookie *http.Cookie) {
	http.SetCookie(ctx.GinContext.Writer, cookie)
}

func (ctx *GinRequestContext) Cookie(name string) (string, error) {
	return ctx.GinContext.Cookie(name)
}

func (ctx *GinRequestContext) ClientIP() string {
	return ctx.GinContext.ClientIP()
}

// Factory function for creating Gin routers
var NewRouter = NewGinRouter