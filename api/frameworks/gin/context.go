package gin

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	gstartAuth "github.com/kod2ulz/gostart/auth"
	"github.com/kod2ulz/gostart/contracts"
)

// GinRequestContext implements the contracts.RequestContext interface for gin.
type GinRequestContext struct {
	ctx *gin.Context
}

// NewRequestContext creates a new GinRequestContext from a gin.Context.
func NewRequestContext(c *gin.Context) contracts.RequestContext {
	return &GinRequestContext{c}
}

// Query returns the value of a URL query parameter.
func (g *GinRequestContext) Query(key string, defaultValue ...string) contracts.Value {
	if val := g.ctx.Query(key); val != "" {
		return contracts.Value(val)
	} else if len(defaultValue) > 0 {
		return contracts.Value(defaultValue[0])
	}
	return ""
}

// Param returns the value of a URL path parameter.
func (g *GinRequestContext) Param(key string, defaultValue ...string) contracts.Value {
	if val := g.ctx.Param(key); val != "" {
		return contracts.Value(val)
	} else if len(defaultValue) > 0 {
		return contracts.Value(defaultValue[0])
	}
	return ""
}

// Header returns the value of an HTTP header.
func (g *GinRequestContext) Header(key string) string {
	return g.ctx.Request.Header.Get(key)
}

// ShouldBindJSON binds the request body to the given object.
func (g *GinRequestContext) ShouldBindJSON(obj interface{}) error {
	return g.ctx.ShouldBindJSON(obj)
}

// Context returns the underlying standard Go context.Context.
func (g *GinRequestContext) Context() context.Context {
	return g.ctx
}

// RequestID returns the unique identifier for this request.
// It checks the gin.Context first (set by middleware), then falls back to headers.
func (g *GinRequestContext) RequestID() string {
	// First try to get from context (set by logging middleware)
	if requestID, exists := g.ctx.Get("request_id"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}

	// Fallback to checking headers directly
	if requestID := g.ctx.GetHeader("X-Request-Id"); requestID != "" {
		return requestID
	}
	// Try alternative header casings
	if requestID := g.ctx.GetHeader("X-Request-ID"); requestID != "" {
		return requestID
	}
	if requestID := g.ctx.GetHeader("Request-Id"); requestID != "" {
		return requestID
	}

	// Last resort: generate a new ID (shouldn't normally happen if middleware is running)
	return "unknown"
}

// Request returns the underlying HTTP request.
func (g *GinRequestContext) Request() *http.Request {
	return g.ctx.Request
}

// Value returns the value associated with this context for key.
func (g *GinRequestContext) Value(key interface{}) interface{} {
	return g.ctx.Value(key)
}

// Set sets a value in the gin context.
func (g *GinRequestContext) Set(key string, value interface{}) {
	g.ctx.Set(key, value)
}

// GinContext returns the underlying gin.Context.
func (g *GinRequestContext) GinContext() *gin.Context {
	return g.ctx
}

// GetUser retrieves the authenticated user from context.
// Returns the user stored in context by authentication middleware.
// Returns nil if no user is authenticated or user is not found in context.
func (g *GinRequestContext) GetUser() any {
	if user := g.ctx.Value(gstartAuth.ContextAuthUserKey); user != nil {
		return user
	}
	return nil
}
