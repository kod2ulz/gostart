package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/contracts"
)

var (
	// Standard errors for user context operations
	ErrNoContext      = errors.New("context is nil")
	ErrNoUserInContext = errors.New("no user in context")
	ErrInvalidUserType = errors.New("invalid user type in context")
)

const (
	// ContextAuthUserKey is the recommended key for storing authenticated users in context
	// Projects can use this constant or define their own
	ContextAuthUserKey = "auth.User"
)

// ============================================
// Minimal User Interface
// ============================================

// User represents a minimal authenticated user interface
// This is the non-generic version for backward compatibility and simple use cases
//
// Note: Use GetID() instead of ID() to avoid conflicts with struct fields named "ID"
type User interface {
	GetID() uuid.UUID
}

// SessionUser is a generic user interface with a typed ID
// This does NOT embed User to avoid type conflicts
// Use this for type-safe user retrieval with custom ID types (string, int, int64, etc.)
type SessionUser[ID comparable] interface {
	GetID() ID
}

// ============================================
// Generic Context Helpers
// ============================================

// GetUser retrieves a User from context using the standard context key
// This is a generic helper that can be used by any project
func GetUser(ctx context.Context) (User, error) {
	if ctx == nil {
		return nil, ErrNoContext
	}

	val := ctx.Value(ContextAuthUserKey)
	if val == nil {
		return nil, ErrNoUserInContext
	}

	user, ok := val.(User)
	if !ok {
		return nil, ErrInvalidUserType
	}

	return user, nil
}

// GetSessionUser retrieves a typed SessionUser from context
// Example: user, err := auth.GetSessionUser[uuid.UUID, MyUser](ctx.Context())
func GetSessionUser[ID comparable, U SessionUser[ID]](ctx context.Context) (U, error) {
	var zero U
	if ctx == nil {
		return zero, ErrNoContext
	}

	val := ctx.Value(ContextAuthUserKey)
	if val == nil {
		return zero, ErrNoUserInContext
	}

	user, ok := val.(U)
	if !ok {
		return zero, ErrInvalidUserType
	}

	return user, nil
}

// ============================================
// Gin Adapter (for convenience)
// ============================================

// ginContextAdapter adapts gin.Context to contracts.RequestContext
type ginContextAdapter struct {
	ctx *gin.Context
}

func (g *ginContextAdapter) Query(key string, defaultValue ...string) contracts.Value {
	if val := g.ctx.Query(key); val != "" {
		return contracts.Value(val)
	} else if len(defaultValue) > 0 {
		return contracts.Value(defaultValue[0])
	}
	return ""
}

func (g *ginContextAdapter) Param(key string, defaultValue ...string) contracts.Value {
	if val := g.ctx.Param(key); val != "" {
		return contracts.Value(val)
	} else if len(defaultValue) > 0 {
		return contracts.Value(defaultValue[0])
	}
	return ""
}

func (g *ginContextAdapter) Header(key string) string {
	return g.ctx.Request.Header.Get(key)
}

func (g *ginContextAdapter) ShouldBindJSON(obj interface{}) error {
	return g.ctx.ShouldBindJSON(obj)
}

func (g *ginContextAdapter) Context() context.Context {
	return g.ctx
}

func (g *ginContextAdapter) RequestID() string {
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
	if requestID := g.ctx.GetHeader("X-Request-ID"); requestID != "" {
		return requestID
	}
	if requestID := g.ctx.GetHeader("Request-Id"); requestID != "" {
		return requestID
	}
	return "unknown"
}

func (g *ginContextAdapter) Request() *http.Request {
	return g.ctx.Request
}

func (g *ginContextAdapter) Value(key interface{}) interface{} {
	return g.ctx.Value(key)
}

func (g *ginContextAdapter) Set(key string, value interface{}) {
	g.ctx.Set(key, value)
}

// ============================================
// Example Middleware Patterns (for documentation)
// ============================================

// The following are EXAMPLE patterns showing how to implement middleware in your project.
// These should be copied and customized in your own codebase, not used directly from gostart.

/*
// Example 1: Simple authentication middleware
// In your project (e.g., internal/middleware/auth.go):

import (
	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/auth"
	"github.com/kod2ulz/gostart/contracts"
	"github.com/yourproject/yourauth"
)

type TokenVerifier struct {
	service yourauth.TokenService
}

func (v *TokenVerifier) VerifyToken(ctx contracts.RequestContext, req yourauth.TokenRequest) (yourauth.User, error) {
	return v.service.Verify(ctx, req.Token, req.Realm)
}

func WithUser(verifier *TokenVerifier) api.MiddlewareFunc {
	return func(ctx contracts.RequestContext) (bool, error) {
		var req yourauth.TokenRequest
		loaded, err := req.RequestLoad(ctx)
		if err != nil {
			return false, err
		}

		req, ok := loaded.(yourauth.TokenRequest)
		if !ok {
			return false, fmt.Errorf("invalid token request type")
		}

		user, err := verifier.VerifyToken(ctx, req)
		if err != nil {
			return false, err
		}

		if ctxSetter, ok := ctx.(interface{ Set(string, interface{}) }); ok {
			ctxSetter.Set(auth.ContextAuthUserKey, user)
		}

		return true, nil
	}
}

// Example 2: Authorization middleware with roles
func WithRoles(roles ...string) api.MiddlewareFunc {
	return func(ctx contracts.RequestContext) (bool, error) {
		user, err := auth.GetUser(ctx.Context())
		if err != nil {
			return false, err
		}

		// Type assert to your concrete user type that has HasRole method
		if typedUser, ok := user.(interface{ HasRole(...string) bool }); ok {
			if !typedUser.HasRole(roles...) {
				return false, fmt.Errorf("insufficient roles")
			}
		}

		return true, nil
	}
}

// Usage in your routes:
//
// router := app.R()
// verifier := &TokenVerifier{service: tokenService}
//
// router.Group("/api", auth.WithUser(verifier)).
//     GET("/admin", auth.WithRoles("admin"), adminHandler).
//     GET("/users", auth.WithPermissions("user.list"), usersHandler)
*/
