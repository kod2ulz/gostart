# Panic Recovery Configuration

The router provides flexible panic recovery options that work across different framework implementations.

## Default Recovery (Recommended)

Each router adapter uses its framework's default recovery:

```go
// Gin router uses gin.Recovery() by default
config := &api.RouterConfig{
    EnableRecovery: true,
}

router, err := gin.NewGinRouter(config)
```

## Override with Custom Recovery Handler

You can provide your own panic recovery handler that will be used instead of the framework default:

```go
import "log/slog"
import "runtime/debug"

config := &api.RouterConfig{
    EnableRecovery: true,
    PanicRecovery: func(panicValue any, stack []byte) {
        slog.Error("Handler panic recovered",
            "panic", panicValue,
            "stack", string(stack),
        )
        // Send to Sentry, Datadog, etc.
    },
}

// The gin adapter will wrap this for gin.CustomRecovery()
router, err := gin.NewGinRouter(config)
```

## Use Native Framework Middleware

You can pass native framework middleware directly:

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/gin-contrib/cors"
)

config := &api.RouterConfig{
    EnableRecovery: false, // Disable built-in recovery
    NativeMiddleware: []any{
        cors.New(cors.Config{
            AllowOrigins:     []string{"*"},
            AllowMethods:     []string{"GET", "POST"},
            AllowHeaders:     []string{"Origin", "Content-Type"},
            AllowCredentials: true,
        }),
        gin.Recovery(), // Use gin's default recovery
        gin.Logger(),    // Use gin's logger
    },
}

router, err := gin.NewGinRouter(config)
```

## Custom Gin Recovery

Use `gin.CustomRecovery()` for custom panic handling:

```go
import (
    "github.com/gin-gonic/gin"
    "log/slog"
    "runtime/debug"
)

config := &api.RouterConfig{
    EnableRecovery: false,
    NativeMiddleware: []any{
        gin.CustomRecovery(func(c *gin.Context, recovered any) {
            slog.Error("Panic recovered",
                "path", c.Request.URL.Path,
                "panic", recovered,
                "stack", string(debug.Stack()),
            )
            c.JSON(500, gin.H{
                "error": "Internal server error",
            })
        }),
        cors.New(cors.Config{
            AllowOrigins: []string{"*"},
        }),
    },
}

router, err := gin.NewGinRouter(config)
```

## Mixing Approaches

You can mix native middleware with custom recovery:

```go
config := &api.RouterConfig{
    EnableRecovery: false, // Don't use built-in
    NativeMiddleware: []any{
        // Use your custom recovery
        gin.CustomRecovery(myCustomRecoveryFunc),

        // Add CORS
        cors.New(cors.Config{AllowOrigins: []string{"*"}}),

        // Add other native middleware
        gin.Logger(),
    },
    // CustomMiddleware for your framework-agnostic middleware
    CustomMiddleware: []api.MiddlewareFunc{
        authMiddleware,
        loggingMiddleware,
    },
}

router, err := gin.NewGinRouter(config)
```

## Framework-Agnostic Custom Recovery

For framework-agnostic panic handling that works across different router implementations:

```go
// Define a custom panic handler
func myPanicHandler(panicValue any, stack []byte) {
    // Log to your monitoring system
    if logger != nil {
        logger.Error("Handler panic",
            "panic", panicValue,
            "stack", string(stack),
        )
    }

    // Send to external monitoring
    // sendToSentry(panicValue, stack)
    // sendToDatadog(panicValue, stack)
}

// Use it in config
config := &api.RouterConfig{
    EnableRecovery: true,
    PanicRecovery:   myPanicHandler,
}

// This will work with gin, echo, fiber, or any other router adapter
router, err := gin.NewGinRouter(config)
```

## How Different Router Adapters Handle Recovery

### Gin Router
- **Default**: Uses `gin.Recovery()`
- **With PanicRecovery**: Uses `gin.CustomRecovery(yourHandler)`
- **With NativeMiddleware**: Uses whatever you pass (gin.Recovery, gin.CustomRecovery, etc.)

### Future Echo/Fiber Adapters
- **Default**: Would use `echo.Recover()` or `fiber.Recover()`
- **With PanicRecovery**: Would wrap it appropriately for that framework
- **With NativeMiddleware**: Would pass middleware to the framework's Use() method

## Recommendation

**For most applications**, use the simple approach:

```go
config := &api.RouterConfig{
    EnableRecovery:   true,
    EnableLogging:    true,
    CustomMiddleware: []api.MiddlewareFunc{
        yourMiddleware,
    },
}
```

This gives you:
- ✅ Default framework recovery (gin.Recovery())
- ✅ Framework-agnostic middleware support
- ✅ Automatic panic recovery for all handlers
- ✅ Works across different router implementations

**For advanced use cases**, provide your own recovery handler:

```go
config := &api.RouterConfig{
    EnableRecovery: true,
    PanicRecovery: func(panicValue any, stack []byte) {
        // Custom logging, monitoring, alerts
    },
}
```

**For complete control**, use native middleware:

```go
config := &api.RouterConfig{
    EnableRecovery: false,
    NativeMiddleware: []any{
        gin.CustomRecovery(yourFunc),
        cors.New(...),
        gin.Logger(),
    },
}
```

## Summary

- **Default**: Framework's native recovery (e.g., `gin.Recovery()`)
- **Override**: Provide `PanicRecovery` function for framework-agnostic custom handling
- **Full Control**: Use `NativeMiddleware` to pass framework-specific middleware directly
- **All approaches work**: Handler chaining and route-specific middleware are always protected
