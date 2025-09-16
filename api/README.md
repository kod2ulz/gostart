# API Package

The `api` package provides a highly productive, opinionated toolkit for building robust, consistent, and framework-adaptable JSON APIs in Go with pluggable web framework support.

## Goal

The primary goal of this package is to drastically reduce boilerplate and enforce a standardized, scalable pattern for API development. It handles the repetitive tasks of the request/response lifecycle so developers can focus purely on business logic.

## Core Concepts

### 1. Framework-Agnostic Router Interface

The package provides a unified router interface that supports multiple web frameworks through a pluggable architecture. Currently supports Gin with extensible architecture for Echo, Fiber, and standard `net/http`.

```go
// Initialize your preferred framework
gin.Setup()

// Use the unified router interface
router := app.R()
router.GET("/users", api.Handler[ListRequest, UserResponse](userService.ListUsers))
```

### 2. Simplified Type-Safe Handlers

The package provides generic, type-safe handler functions that eliminate boilerplate code:

- `Handler[P, T]`: For single object responses with request parameters
- `ListHandler[P, T]`: For list responses with pagination
- `FileHandler[P]`: For file responses

```go
// Single object response with typed parameters
router.GET("/users/:id", api.Handler[GetUserRequest, User](func(ctx context.Context, req GetUserRequest) (User, error) {
    return userService.GetUser(req.ID)
}))

// List response with pagination
router.GET("/users", api.ListHandler[ListUsersRequest, User](func(ctx context.Context, req ListUsersRequest) ([]User, *int64, error) {
    return userService.ListUsers(req)
}))

// File response
router.GET("/files/:id", api.FileHandler[GetFileRequest](func(ctx context.Context, req GetFileRequest) (FileResponse, error) {
    return fileService.GetFile(req.ID)
}))
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

## Roadmap

Future enhancements for this package include:

- [x] **Framework Agnosticism**: Complete framework-agnostic design allowing use with Gin, Echo, Fiber, or standard `net/http`.
- [ ] **Enhanced Error Parsing**: Deeper inspection of database and validator errors to provide even more specific and helpful error messages (e.g., "user with this email already exists" from a SQL unique constraint violation).
- [ ] **Automated API Documentation**: Structure the API definitions in a way that enables the automatic generation of OpenAPI (Swagger) specifications.
- [ ] **Additional Framework Implementations**: Built-in support for Echo, Fiber, and other popular frameworks.
