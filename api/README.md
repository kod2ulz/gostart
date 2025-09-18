# API Package

The `api` package provides a framework-agnostic, highly productive toolkit for building robust, consistent JSON APIs in Go. It implements a clean separation between core application logic and web framework specifics.

## Goal

The primary goal is to provide a standardized, scalable pattern for API development that drastically reduces boilerplate while maintaining complete framework agnosticism. Developers can write business logic once and easily switch between web frameworks (Gin, Echo, Fiber, etc.) without changing their core application code.

## Core Concepts

### 1. Framework-Agnostic Router Interface

The package provides a unified router interface that supports multiple web frameworks through a pluggable architecture. Currently supports Gin with extensible architecture for Echo, Fiber, and standard `net/http`.

```go
// Initialize your preferred framework
gin.Setup()

// Use the unified router interface
router := app.R()
router.GET("/users", api.JSONHandler[[]User](userService.ListUsers))
```

### 2. Simplified Type-Safe Handlers

The package provides generic, type-safe handler functions that eliminate boilerplate code:

- `JSONHandler[T]`: For JSON responses with automatic request loading
- File handlers: For non-JSON responses (io.Reader, byte data)

```go
// Single object response
router.GET("/users/:id", api.JSONHandler[User](func(ctx contracts.RequestContext) (User, ierrors.Error) {
    var req GetUserRequest

    // Load from context (correct pattern - RequestLoad is called automatically by JSONHandler)
    if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
        return User{}, err
    }

    return userService.GetUser(req.ID)
}))

// List response
router.GET("/users", api.JSONHandler[[]User](func(ctx contracts.RequestContext) ([]User, ierrors.Error) {
    var req ListUsersRequest

    // Load from context (correct pattern)
    if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
        return nil, err
    }

    return userService.ListUsers(req)
}))

// File response for downloads
router.GET("/files/:id", func(ctx contracts.RequestContext) {
    // File handlers can return io.Reader or byte data
    data, filename, err := fileService.GetFile(req.ID)
    if err != nil {
        ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }

    ctx.Header("Content-Disposition", "attachment; filename="+filename)
    ctx.Data(http.StatusOK, "application/octet-stream", data)
})
```

### 3. Unified Response Envelope

All responses use a standardized JSON structure with consistent formatting:

```json
{
  "success": true,
  "type": "User",
  "data": { ... },
  "references": {
    "countries": [...],
    "roles": [...]
  },
  "meta": {
    "total": 100,
    "limit": 10,
    "offset": 0
  },
  "time": 1634567890
}
```

The envelope automatically handles:
- Single objects and lists with pagination
- Reference data at the same level as data (not in meta)
- Error responses with structured error information
- Metadata for pagination and custom data
- Consistent timestamps and response types

### 4. Request Parameter Interface

Request parameters implement the `contracts.RequestParam` interface, providing a standardized way to load and validate request data from various sources (JSON body, URL parameters, headers).

```go
type CreateUserRequest struct {
    api.RequestModal[CreateUserRequest] // Embed for default behavior
    Name  string `json:"name" validate:"required"`
    Email string `json:"email" validate:"required,email"`
}

type GetUserRequest struct {
    api.RequestModal[GetUserRequest]
    ID string `param:"id" validate:"required,uuid"`
}

type ListUsersRequest struct {
    api.ListRequest // Embed for pagination support
    Search string `query:"search"`
    Active bool `query:"active"`
}
```

### 5. Comprehensive Error Handling

Built-in error handling that automatically formats errors into consistent responses:

```go
// Errors are automatically handled and formatted
if err != nil {
    return nil, errors.ServiceErrorBadRequest("Invalid user data")
}
```

### 6. Framework Registration

Easy framework registration system that doesn't imply limited support:

```go
// Register Gin as the framework
gin.Setup()

// Or with custom configuration
gin.SetupWithOptions(func(config *api.RouterConfig) {
    config.AllowOrigins = []string{"*"}
    config.EnableRecovery = true
    config.EnableLogging = true
})
```

## Architecture Status

- [x] **Framework Agnosticism**: Complete framework-agnostic design allowing use with Gin, Echo, Fiber, or standard `net/http`.
- [x] **Comprehensive Test Coverage**: Full test coverage for router initialization, error handling, and file responses.
- [x] **File Handler Support**: Complete support for non-JSON responses including io.Reader and byte data streaming.
- [x] **OpenAPI Integration**: Automatic OpenAPI documentation generation with route registration.

## Future Enhancements

- [ ] **Enhanced Error Parsing**: Deeper inspection of database and validator errors to provide even more specific and helpful error messages.
- [ ] **Additional Framework Implementations**: Built-in support for Echo, Fiber, and other popular frameworks.
- [ ] **Response Streaming**: Enhanced support for streaming responses and real-time data.
