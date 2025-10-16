# Contracts Package

The `contracts` package defines the core interfaces and types that enable framework-agnostic API development in GoStart. These contracts establish the boundaries between application logic and web framework implementations.

## Overview

This package provides the foundational interfaces that allow:
- Framework-agnostic request handling
- Type-safe parameter loading and validation
- Standardized response structures
- Clean separation of concerns

## Core Interfaces

### 1. RequestContext

The `RequestContext` interface abstracts HTTP request data access, making handlers independent of the underlying web framework:

```go
type RequestContext interface {
    // Query parameters with optional defaults
    Query(key string, defaultValue ...string) Value

    // URL path parameters with optional defaults
    Param(key string, defaultValue ...string) Value

    // HTTP headers
    Header(key string) string

    // JSON body binding
    ShouldBindJSON(obj interface{}) error

    // Standard Go context
    Context() context.Context
}
```

**Usage Example:**
```go
func MyHandler(ctx contracts.RequestContext) {
    // Access query parameters
    page := ctx.Query("page", "1").Int()

    // Access path parameters
    id := ctx.Param("id").String()

    // Access headers
    auth := ctx.Header("Authorization")

    // Bind JSON body
    var req MyRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        // handle error
    }
}
```

### 2. RequestParam

The `RequestParam` interface defines how request parameters are loaded and validated:

```go
type RequestParam interface {
    // Validate the request using struct tags
    Validate(ctx RequestContext) error

    // Load and populate from HTTP context
    RequestLoad(ctx RequestContext) (RequestParam, error)

    // Get context storage key
    ContextKey() string

    // Load from standard Go context
    ContextLoad(ctx context.Context) (RequestParam, error)
}
```

**Usage Example:**
```go
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"gte=18"`
}
```

**Pattern 1: Automatic Validation (Recommended)**
```go
// Handler usage - validation is automatic via struct tags
func CreateUserHandler(ctx contracts.RequestContext) (User, ierrors.Error) {
    var req CreateUserRequest

    // Load from context (RequestLoad is called automatically by JSONHandler)
    if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
        return User{}, err
    }

    // Business logic - req is already validated
    return userService.CreateUser(req)
}
```

**Pattern 2: Custom Validation Logic**
```go
// For cases where you need custom validation beyond struct tags
func (r *CreateUserRequest) RequestLoad(ctx contracts.RequestContext) (contracts.RequestParam, error) {
    var out CreateUserRequest

    // Load data using the default loader
    if err := contracts.DefaultRequestLoader(ctx, &out); err != nil {
        return nil, errors.ValidatorError[CreateUserRequest](err)
    }

    // Add custom business validation
    if out.Age < 18 {
        return nil, errors.ValidationFailed[CreateUserRequest](
            fmt.Errorf("user must be at least 18 years old"))
    }

    // Additional custom logic
    if strings.Contains(out.Name, "admin") && out.Email != "admin@example.com" {
        return nil, errors.ValidationFailed[CreateUserRequest](
            fmt.Errorf("admin-related names require admin email"))
    }

    return out, nil
}

// Handler with custom validation
func CreateUserHandlerCustom(ctx contracts.RequestContext) (User, ierrors.Error) {
    var req CreateUserRequest

    // Load from context (uses custom RequestLoad method)
    if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
        return User{}, err
    }

    // Business logic
    return userService.CreateUser(req)
}
```

### 3. Value Type

The `Value` type provides convenient type conversion for parameter values:

```go
type Value string

// Conversion methods
func (v Value) String() string
func (v Value) Int() int
func (v Value) Int64() int64
func (v Value) Float64() float64
func (v Value) Bool() bool
func (v Value) Valid() bool
```

**Usage Example:**
```go
page := ctx.Query("page", "1").Int()        // Default to 1
limit := ctx.Query("limit", "10").Int64()   // Default to 10
active := ctx.Query("active", "false").Bool() // Default to false
```

## Response Types

### Standard Response Structure

The `Response[T]` type provides a standardized JSON response format:

```go
type Response[T any] struct {
    Success    bool           `json:"success"`
    Type       string         `json:"type,omitempty"`
    Error      ierrors.Error  `json:"error,omitempty"`
    Data       interface{}    `json:"data,omitempty"`
    References map[string]any `json:"references,omitempty"`
    Meta       *Metadata      `json:"meta,omitempty"`
    Timestamp  int64          `json:"time,omitempty"`
}
```

**Example Response:**
```json
{
  "success": true,
  "type": "User",
  "data": {
    "id": "123",
    "name": "John Doe",
    "email": "john@example.com"
  },
  "references": {
    "roles": [{"id": "admin", "name": "Administrator"}]
  },
  "meta": {
    "total": 100,
    "page": 1,
    "limit": 10
  },
  "time": 1634567890
}
```

### Response Builders

The package provides helper functions for creating responses:

```go
// Success response
resp := contracts.SuccessResponse(user)

// Error response
resp := contracts.ErrorResponse[User](errors.NotFound("User not found"))

// Response with references
resp := contracts.SuccessResponse(user).
    WithReferences(map[string]any{
        "roles": roles,
        "permissions": permissions,
    })
```

## Metadata Support

The `Metadata` type provides pagination and custom metadata:

```go
type Metadata struct {
    Total    int64             `json:"total,omitempty"`
    Page     int               `json:"page,omitempty"`
    Limit    int               `json:"limit,omitempty"`
    Offset   int               `json:"offset,omitempty"`
    Custom   map[string]any    `json:"custom,omitempty"`
}

// Usage
meta := &contracts.Metadata{
    Total: 100,
    Page: 1,
    Limit: 10,
    Custom: map[string]any{
        "filter": "active",
        "sort": "name",
    },
}
```

## Best Practices

### 1. Request Parameter Design

```go
// Good: Clear validation rules and struct tags
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=2,max=100"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"gte=18,lte=120"`
}

// Handler usage (loading is automatic)
func CreateUserHandler(ctx contracts.RequestContext) (User, ierrors.Error) {
    var req CreateUserRequest

    // Load from context - RequestLoad is handled automatically by the framework
    if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
        return User{}, err
    }

    // Business logic
    return userService.CreateUser(req)
}
```

### 2. Error Handling

```go
func MyHandler(ctx contracts.RequestContext) (User, ierrors.Error) {
    var req MyRequest

    // Load from context (correct pattern)
    if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
        return User{}, err // Error already properly wrapped
    }

    // Business logic
    user, err := userService.Create(req)
    if err != nil {
        return User{}, err // Return ierrors.Error directly
    }

    return user, nil
}
```

### 3. Response Formatting

```go
// Single object
return contracts.SuccessResponse(user)

// List with metadata
return contracts.SuccessResponse(users).
    WithMetadata(&contracts.Metadata{
        Total: totalCount,
        Page: req.Page,
        Limit: req.Limit,
    })

// With references
return contracts.SuccessResponse(user).
    WithReferences(map[string]any{
        "roles": userRoles,
        "departments": departments,
    })
```

## Framework Integration

The contracts are designed to work seamlessly with different web frameworks:

### Gin Framework Example
```go
// Gin adapter implements RequestContext
type ginRequestContext struct {
    ctx *gin.Context
}

func (g *ginRequestContext) Query(key string, defaultValue ...string) contracts.Value {
    if val := g.ctx.Query(key); val != "" {
        return contracts.Value(val)
    }
    if len(defaultValue) > 0 {
        return contracts.Value(defaultValue[0])
    }
    return ""
}

// Usage in Gin handlers
router.GET("/users/:id", func(c *gin.Context) {
    ctx := &ginRequestContext{ctx: c}
    user, err := MyHandler(ctx)
    // Handle response
})
```

## Testing

The interfaces make testing straightforward:

```go
func TestMyHandler(t *testing.T) {
    // Mock RequestContext
    mockCtx := &mockRequestContext{
        queryParams: map[string]string{"page": "1"},
        pathParams:  map[string]string{"id": "123"},
    }

    result, err := MyHandler(mockCtx)

    // Assertions
    assert.NoError(t, err)
    assert.Equal(t, "John Doe", result.Name)
}
```

## Migration from Old Architecture

If migrating from the old architecture:

1. **Replace `gin.Context` with `contracts.RequestContext`**
2. **Update handler signatures to return `(T, ierrors.Error)`**
3. **Use `contracts.RequestModal[T]` for parameter loading**
4. **Update response construction to use `contracts.Response[T]`**

The contracts package ensures your business logic remains framework-agnostic while providing all the tools needed for robust API development.