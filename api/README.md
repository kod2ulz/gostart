# API Package

The `api` package provides a highly productive, opinionated toolkit for building robust, consistent, and framework-adaptable JSON APIs in Go with pluggable web framework support.

## Goal

The primary goal of this package is to drastically reduce boilerplate and enforce a standardized, scalable pattern for API development. It handles the repetitive tasks of the request/response lifecycle so developers can focus purely on business logic.

## Core Concepts

### 1. Framework-Agnostic Router Interface

The package provides a unified router interface that supports multiple web frameworks through a pluggable architecture. Currently supports Gin with plans for Echo, Fiber, and standard `net/http`.

```go
// Initialize your preferred framework
gin.Setup()

// Use the unified router interface
router := app.R()
router.GET("/users", api.Handler[UserResponse](userService.ListUsers))
```

### 2. Simplified Type-Safe Handlers

The package provides generic, type-safe handler functions that eliminate boilerplate code:

- `Handler[T]`: For single object responses
- `ListHandler[T]`: For list responses with pagination
- `JSONHandler[T]`: For simple JSON responses without request parameters
- `StreamHandler[T]`: For streaming responses
- `FileHandler`: For file responses
- `DownloadHandler`: For file downloads

```go
// Simple single object response
router.GET("/users/:id", api.Handler(func(ctx contracts.RequestContext, param contracts.RequestParam) (User, ierrors.Error) {
    id := ctx.Param("id")
    return userService.GetUser(id)
}))

// List response with pagination
router.GET("/users", api.ListHandler(func(ctx contracts.RequestContext, param contracts.RequestParam) ([]User, *int64, ierrors.Error) {
    return userService.ListUsers(param)
}))

// Simple JSON response without request parameters
router.GET("/health", api.JSONHandler(func(ctx contracts.RequestContext) (Health, ierrors.Error) {
    return healthService.Check()
}))
```

### 3. Unified Response Envelope

All responses use a standardized JSON structure with consistent formatting:

```json
{
  "success": true,
  "type": "User",
  "data": { ... },
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
- Error responses with structured error information
- Metadata and reference data
- Consistent timestamps and response types

### 4. Request Parameter Interface

Request parameters implement the `contracts.RequestParam` interface, providing a standardized way to load and validate request data from various sources (JSON body, URL parameters, headers).

```go
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required"`
    Email string `json:"email" validate:"required,email"`
    contracts.RequestModal[CreateUserRequest] // Embed for default behavior
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

## Roadmap

Future enhancements for this package include:

- **Enhanced Error Parsing**: Deeper inspection of database and validator errors to provide even more specific and helpful error messages (e.g., "user with this email already exists" from a SQL unique constraint violation).
- **Framework Agnosticism**: The long-term vision is to leverage the `RequestParam` interface to allow the request/response lifecycle to be used with other web frameworks like Fiber, Echo, or the standard `net/http` library.
- **Automated API Documentation**: Structure the API definitions in a way that enables the automatic generation of OpenAPI (Swagger) specifications.
