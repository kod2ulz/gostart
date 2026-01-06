# Panic Recovery Implementation

## Problem Identified

The server was crashing when panics occurred in handlers because the handler chaining logic created plain Go closures that bypassed Gin's built-in recovery middleware.

## Solution Implemented

### 1. **Centralized Panic Recovery** (`api/frameworks/gin/router.go:328-347`)

Added a `handlePanic()` method that provides centralized panic recovery with support for custom panic handlers:

```go
func (r *GinRouter) handlePanic(c *gin.Context, fn func()) {
    defer func() {
        if recovered := recover(); recovered != nil {
            // Call custom panic handler if provided
            if r.config != nil && r.config.PanicRecovery != nil {
                stack := debug.Stack()
                r.config.PanicRecovery(recovered, stack)
            }

            // Return error response to client
            c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]interface{}{
                "error":   "Internal server error",
                "message": fmt.Sprintf("Panic recovered: %v", recovered),
            })
        }
    }()

    fn()
}
```

### 2. **Panic Recovery in Handler Chaining** (`router.go:270-287`)

When chaining additional handlers with `WithHandler()`, each chained handler now has its own panic recovery:

```go
wrapped := func(ctx contracts.RequestContext) {
    defer func() {
        if recovered := recover(); recovered != nil {
            // Call custom panic handler if provided
            if r.config != nil && r.config.PanicRecovery != nil {
                stack := debug.Stack()
                r.config.PanicRecovery(recovered, stack)
            }

            // Return error response to client
            if requestCtx, ok := ctx.(*RequestContext); ok {
                requestCtx.ctx.AbortWithStatusJSON(http.StatusInternalServerError, map[string]interface{}{
                    "error":   "Internal server error",
                    "message": fmt.Sprintf("Panic recovered: %v", recovered),
                })
            }
        }
    }()

    currentHandler(ctx)
    nextHandler(ctx)
}
```

### 3. **Panic Recovery in Middleware Wrapping** (`router.go:575-599`)

Route-specific middleware and handlers are wrapped with panic recovery:

```go
func (r *GinRouter) wrapHandlerWithMiddleware(handler api.HandlerFunc, middlewares []api.MiddlewareFunc) gin.HandlerFunc {
    return func(c *gin.Context) {
        r.handlePanic(c, func() {
            // Execute middleware and handler
            // ...
        })
    }
}
```

### 4. **Custom Panic Handler Configuration** (`api/router.go:121`)

Added `PanicRecovery` field to `RouterConfig`:

```go
type RouterConfig struct {
    // ... other fields
    PanicRecovery func(c any, stack []byte) // Custom panic recovery handler
}
```

## Usage

### Default Panic Recovery

Out of the box, all panics are automatically caught and return a 500 error:

```go
// This will NOT crash the server
router.GET("/panic", func(ctx contracts.RequestContext) {
    panic("something went wrong")
})
```

Response:
```json
{
  "error": "Internal server error",
  "message": "Panic recovered: something went wrong"
}
```

### Custom Panic Handler

You can provide a custom panic handler to log panics, send alerts, etc.:

```go
import (
    "log/slog"
    "runtime/debug"
    "github.com/kod2ulz/gostart/api"
)

func main() {
    config := &api.RouterConfig{
        EnableRecovery: true,
        PanicRecovery: func(panicValue any, stack []byte) {
            slog.Error("Handler panic recovered",
                "panic", panicValue,
                "stack", string(stack),
            )
            // You could also send to Sentry, Datadog, etc.
        },
    }

    router, err := gin.NewGinRouter(config)
    // ...
}
```

### Multiple Layers of Protection

Panic recovery now works at multiple levels:

1. **Handler Chaining**: Each chained handler has its own recovery
2. **Middleware**: Route-specific middleware is protected
3. **Main Handler**: The final handler is protected
4. **Global Recovery**: Gin's global recovery middleware (if enabled) provides a final fallback

## Example: Testing Panic Recovery

```go
func TestPanicRecovery(t *testing.T) {
    // Create a handler that panics
    handler := func(ctx contracts.RequestContext) {
        panic("intentional panic for testing")
    }

    router := setupTestRouter()
    router.GET("/test", handler)

    // Make request
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/test", nil)
    router.ServeHTTP(w, req)

    // Should return 500, not crash
    assert.Equal(t, 500, w.Code)
    assert.Contains(t, w.Body.String(), "Panic recovered")
}
```

## Key Benefits

1. ✅ **Server Never Crashes**: All panics are caught and recovered
2. ✅ **Consistent Error Responses**: Clients always get proper JSON error responses
3. ✅ **Customizable**: Provide your own panic handler for logging/alerting
4. ✅ **Stack Traces**: Custom handlers receive full stack traces
5. ✅ **Works at All Levels**: Handler chaining, middleware, and main handlers are all protected
6. ✅ **Backward Compatible**: Existing code continues to work without changes

## Migration Notes

- If you were using `gin.Recovery()` middleware before, you can remove it as recovery is now built-in
- To keep using `gin.Recovery()`, set `EnableRecovery: true` in RouterConfig
- Custom panic handlers receive the panic value and full stack trace for debugging
