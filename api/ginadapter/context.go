package ginadapter

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/kod2ulz/gostart/contracts"
)

// GinRequestContext implements the contracts.RequestContext interface for gin.
type GinRequestContext struct {
	GinContext *gin.Context
}

// NewRequestContext creates a new GinRequestContext from a gin.Context.
func NewRequestContext(c *gin.Context) contracts.RequestContext {
	return &GinRequestContext{c}
}

// Query returns the value of a URL query parameter.
func (g *GinRequestContext) Query(key string, defaultValue ...string) contracts.Value {
	if val := g.GinContext.Query(key); val != "" {
		return contracts.Value(val)
	} else if len(defaultValue) > 0 {
		return contracts.Value(defaultValue[0])
	}
	return ""
}

// Param returns the value of a URL path parameter.
func (g *GinRequestContext) Param(key string, defaultValue ...string) contracts.Value {
	if val := g.GinContext.Param(key); val != "" {
		return contracts.Value(val)
	} else if len(defaultValue) > 0 {
		return contracts.Value(defaultValue[0])
	}
	return ""
}

// Header returns the value of an HTTP header.
func (g *GinRequestContext) Header(key string) string {
	return g.GinContext.Request.Header.Get(key)
}

// ShouldBindJSON binds the request body to the given object.
func (g *GinRequestContext) ShouldBindJSON(obj interface{}) error {
	return g.GinContext.ShouldBindJSON(obj)
}

// Context returns the underlying standard Go context.Context.
func (g *GinRequestContext) Context() context.Context {
	return g.GinContext
}

// Value returns the value associated with this context for key.
func (g *GinRequestContext) Value(key interface{}) interface{} {
	return g.GinContext.Value(key)
}

// Set sets a value in the gin context.
func (g *GinRequestContext) Set(key string, value interface{}) {
	g.GinContext.Set(key, value)
}