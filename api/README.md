# API Package

This package provides a robust, framework-agnostic API development layer for Go applications that eliminates boilerplate and maximizes developer productivity. It offers clean interfaces for building HTTP APIs that can work with multiple web frameworks while maintaining type safety and consistency.

## Why You'll Love Building APIs with GoStart

The API package transforms how you build web APIs in Go by addressing the common pain points that slow down development:

### 🚀 **Zero-Configuration Philosophy**
- **Automatic OpenAPI Documentation**: Generate comprehensive API docs without writing a single annotation
- **Smart Request Processing**: Automatic parameter loading from JSON, query params, URL params, and headers
- **Zero-Boilerplate Responses**: Standardized JSON response format with zero setup
- **Framework Independence**: Write once, deploy with Gin, Echo, Fiber, or standard net/http

### 🎯 **Type-Safe by Design**
- **Compile-Time Validation**: Leverage Go's generics to catch errors before runtime
- **Generic Handler Patterns**: `Handler[T]`, `ListHandler[T]`, `StreamHandler[T]` for every use case
- **Automatic Request/Response Marshalling**: Focus on business logic, not serialization
- **Strongly-Typed Configuration**: Catch configuration errors at compile time

### 🔧 **Productivity-Boosting Features**
- **Cleaner Syntax**: Express complex API logic with minimal, readable code
- **Automatic Error Handling**: Built-in error responses with proper HTTP status codes
- **Middleware Ecosystem**: Plug-in logging, authentication, CORS, and more
- **Framework Registry**: Easily add support for new web frameworks

### 🏗️ **Production-Ready Architecture**
- **Layered Design**: Clean separation between business logic and framework details
- **Extensible by Default**: Plugin architecture for custom middleware and frameworks
- **Performance-Optimized**: Minimal overhead with efficient request handling
- **Enterprise-Grade**: Built for real-world applications with comprehensive testing

## Core Architecture

The package follows a layered architecture that separates concerns cleanly:

```
contracts.RequestContext (interface)
    ↓
api.RequestContext (extended interface)
    ↓
api/frameworks/gin/context.go (Gin implementation)
```

This design ensures your business logic remains framework-agnostic while providing rich functionality when needed.

### 1. **Framework-Agnostic Router Interface**
Write handlers once using the `RequestContext` interface, and they work with any supported framework. Currently supports Gin with extensible architecture for Echo, Fiber, and standard `net/http`.

### 2. **Simplified Type-Safe Handlers**
Generic handler patterns eliminate boilerplate while maintaining type safety:
- `Handler[T]` - Single object responses
- `ListHandler[T]` - Paginated list responses
- `StreamHandler[T]` - Streaming responses
- `FileHandler` - File upload/download
- `JSONHandler[T]` - Simple JSON responses

### 3. **Type-Safe Request/Response Processing**
Automatic parameter loading and response serialization with compile-time validation. The `RequestModal[T]` system handles JSON, query parameters, URL parameters, and headers seamlessly.

### 4. **Automatic Zero-Configuration Documentation**
Comprehensive OpenAPI 3.0 documentation generated automatically from your code structure. No annotations needed - just write Go code and get interactive API docs.

### 5. **Extensible Middleware System**
Plugin architecture for middleware that works across all frameworks. Built-in logging, authentication, and CORS with easy extension points.

## Getting Started

Let's build a complete user management API to see the power of GoStart in action. This example demonstrates the zero-boilerplate approach that makes GoStart so productive.

### Setting Up Your Application

First, initialize the framework and create your application. GoStart handles all the setup automatically:

```go
package main

import (
    "github.com/kod2ulz/gostart/api"
    "github.com/kod2ulz/gostart/api/frameworks/gin"
    "github.com/kod2ulz/gostart/app"
    "github.com/kod2ulz/gostart/contracts"
)

// Define your request/response types
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=2,max=100"`
    Email string `json:"email" validate:"required,email"`
}

type User struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

func main() {
    // Setup framework and initialize application
    gin.Setup()
    application := app.Init(app.WithHeartbeatHandlers())
    router := application.R()

    // Create user endpoint
    router.POST("/users", api.JSONHandler[User](func(ctx contracts.RequestContext) (User, error) {
        var req CreateUserRequest

        // Automatic loading from JSON body with validation
        if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
            return User{}, err
        }

        // Business logic
        user := User{
            ID:        generateID(),
            Name:      req.Name,
            Email:     req.Email,
            CreatedAt: time.Now(),
        }

        return userService.Create(user)
    }))

    // Get user by ID
    router.GET("/users/:id", api.JSONHandler[User](func(ctx contracts.RequestContext) (User, error) {
        id := ctx.Param("id")
        user, err := userService.GetByID(id)
        if err != nil {
            return User{}, err
        }
        return user, nil
    }))

    application.Run()
}
```

## What's Happening Here?

The `JSONHandler[T]` wrapper provides:
- **Automatic request loading** from JSON body, query parameters, and URL parameters
- **Type-safe responses** with generic type parameters
- **Standardized error handling** with proper HTTP status codes
- **Consistent JSON response format** with success indicators and metadata

The resulting JSON response follows this structure:
```json
{
  "success": true,
  "type": "User",
  "data": {
    "id": "123",
    "name": "John Doe",
    "email": "john@example.com"
  },
  "time": 1634567890
}
```

## Key Features

### Request Parameter Loading

The package supports loading request parameters from multiple sources using struct tags:

```go
type SearchRequest struct {
    // From URL path
    ID string `param:"id"`

    // From query parameters
    Query string `query:"q"`
    Limit int    `query:"limit" validate:"min=1,max=100"`

    // From JSON body
    Filters []string `json:"filters"`

    // From headers
    Authorization string `header:"Authorization"`
}

router.GET("/search/:id", api.JSONHandler[SearchResult](func(ctx contracts.RequestContext) (SearchResult, error) {
    var req SearchRequest

    // Load from all sources automatically
    if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
        return SearchResult{}, err
    }

    return searchService.Search(req)
}))
```

### Handler Types

The package provides several type-safe handler wrappers for different use cases:

```go
// Single object response
router.POST("/users", api.JSONHandler[User](func(ctx contracts.RequestContext) (User, error) {
    var req CreateUserRequest
    if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
        return User{}, err
    }
    return userService.Create(req)
}))

// Array response with pagination
router.GET("/users", api.ListHandler[User](func(ctx contracts.RequestContext) ([]User, error) {
    return userService.GetAll()
}))

// File download
router.GET("/files/:id", api.FileHandler(func(ctx contracts.RequestContext) (string, error) {
    fileID := ctx.Param("id")
    return fileService.Download(fileID)
}))

// Stream response
router.GET("/export", api.StreamHandler(func(ctx contracts.RequestContext) (<-chan User, error) {
    return userService.StreamAllUsers()
}))
```

### Response Helpers

Use the package-level response functions for standardized responses:

```go
// Success response
err := api.SuccessResponse(ctx, user)

// List response with pagination
err := api.ListResponse(ctx, users, &total, &limit, &offset)

// Error response
err := api.ErrorResponse(ctx, "USER_NOT_FOUND", "User not found", http.StatusNotFound)

// Validation error
err := api.ValidationError(ctx, "Invalid input", fieldErrors)
```

### Middleware Architecture

GoStart provides a clean middleware system that works across all frameworks. The middleware architecture is designed to be both powerful and simple:

```go
// Built-in logging middleware with detailed configuration
router.Use(api.LoggingMiddleware(logger, &api.RequestLogConfig{
    LogRequest:  true,
    LogResponse: true,
    LogHeaders:  false,
    LogBody:     false,
    SensitiveHeaders: []string{"Authorization", "Cookie"},
}))

// Authentication middleware using the auth package
router.Use(auth.WithUser[User]())

// Custom middleware with full context access
router.Use(func(ctx contracts.RequestContext) {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        logger.Info("Request processed",
            "method", ctx.Method(),
            "path", ctx.Path(),
            "duration", duration,
            "status", ctx.Status(),
        )
    }()
    ctx.Next()
})
```

### Available Middleware

**✅ Currently Implemented:**
- **Logging Middleware** - Comprehensive request/response logging with configurable sensitivity
- **Authentication Middleware** - Generic user authentication with framework adapters
- **Request ID Generation** - Automatic request tracking and correlation
- **Framework Integration** - Gin's built-in recovery and CORS middleware

**🚧 Planned Enhancements:**
- Rate limiting middleware
- Response caching middleware
- API versioning middleware
- Advanced CORS configuration

## Implementation Status

### ✅ **Fully Implemented Features**
- **Framework-Agnostic Architecture** - Complete `Router` interface with Gin implementation
- **Type-Safe Generic Handlers** - `Handler[T]`, `ListHandler[T]`, `StreamHandler[T]`, `FileHandler`
- **Zero-Configuration OpenAPI Documentation** - Automatic generation with interactive Swagger UI
- **Smart Request Processing** - `RequestModal[T]` with automatic JSON/query/URL/header loading
- **Standardized Response Envelope** - `ResponseEnvelope` with success/error/metadata support
- **Comprehensive Logging Middleware** - Request ID tracking, structured logging, configurable sensitivity
- **Framework Registry System** - Plugin architecture for adding new web frameworks
- **Request Validation** - Automatic validation with detailed error messages
- **Error Handling Integration** - Seamless integration with gostart errors package
- **Gin Framework Integration** - Complete production-ready implementation

### ⚠️ **Partially Implemented**
- **Authentication Middleware** - Basic middleware structure exists, but authentication logic not implemented
- **CORS Configuration** - Basic configuration through environment variables, limited advanced controls
- **File Upload/Download** - Handler interfaces exist, implementation needs verification

### 🚧 **Planned Features (Not Yet Implemented)**
- **Additional Framework Support** - Echo, Fiber, and standard net/http adapters
- **Authentication Implementation** - JWT/PASETO token validation and user management
- **Rate Limiting Middleware** - Configurable request rate limiting
- **Response Caching** - Intelligent caching middleware
- **Advanced CORS Configuration** - More granular CORS controls
- **API Versioning** - Clean version management without code duplication

## Framework Support

**Currently Supported:**
- **Gin** - Full feature parity with production-ready implementation

**Planned Support:**
- **Echo** - Framework adapter in development
- **Fiber** - High-performance framework adapter planned
- **Standard net/http** - Native Go HTTP server support

The framework registry pattern makes it easy to add support for new web frameworks. All handlers written with the `RequestContext` interface work with any supported framework without code changes.

## Framework Support

The package currently supports **Gin** as the primary web framework. The architecture is designed to support additional frameworks through adapters:

```go
// Currently supported
gin.Setup()

// Planned for future implementation
// echo.Setup()
// fiber.Setup()
// http.Setup()
```

All handlers written with the `RequestContext` interface will work with any supported framework without code changes.

## Integration with Other Packages

The API package integrates seamlessly with other GoStart packages:

```go
// Error handling
import "github.com/kod2ulz/gostart/errors"

// Configuration
import "github.com/kod2ulz/gostart/config"

// Logging
import "github.com/kod2ulz/gostart/logr"

// Collections (for data processing)
import "github.com/kod2ulz/gostart/collections"
```

## Best Practices

### Handler Organization
```go
// Organize handlers by feature
type UserHandlers struct {
    userService *UserService
}

func (h *UserHandlers) Create(ctx contracts.RequestContext) (User, error) {
    var req CreateUserRequest
    if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
        return User{}, err
    }
    return h.userService.Create(req)
}

// Register routes
handlers := &UserHandlers{userService: userService}
router.POST("/users", api.JSONHandler[User](handlers.Create))
```

### Error Handling
```go
router.GET("/users/:id", api.JSONHandler[User](func(ctx contracts.RequestContext) (User, error) {
    id := ctx.Param("id")
    user, err := userService.GetByID(id)
    if err != nil {
        return User{}, errors.NotFound("user not found: %s", id)
    }
    return user, nil
}))
```

### Validation
```go
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=2,max=100"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"gte=18,lte=120"`
}

// Validation is automatic
router.POST("/users", api.JSONHandler[User](func(ctx contracts.RequestContext) (User, error) {
    var req CreateUserRequest
    if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
        return User{}, err // Returns 400 with validation errors
    }
    return userService.Create(req)
}))
```

## Learn More

📚 **Detailed guides:**
- [Handler Types](./docs/handlers.md) - Complete handler reference
- [Request Processing](./docs/request-processing.md) - Parameter loading and validation
- [Middleware Patterns](./docs/middleware.md) - Available middleware and custom middleware
- [Framework Integration](./docs/frameworks.md) - Adding support for new frameworks

---

This package provides a solid foundation for building web APIs in Go with an emphasis on clean architecture, type safety, and developer productivity.