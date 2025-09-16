# API Package

The `api` package provides a highly productive, opinionated toolkit for building robust, consistent, and framework-adaptable JSON APIs in Go, with `gin` as the default web framework.

## Goal

The primary goal of this package is to drastically reduce boilerplate and enforce a standardized, scalable pattern for API development. It handles the repetitive tasks of the request/response lifecycle so developers can focus purely on business logic.

## Core Concepts

### 1. The Generic Request Lifecycle

The package automates the entire request lifecycle. When a request hits an endpoint managed by a generic handler, the following steps occur automatically:

1.  **Load Request**: A custom request struct is loaded and populated from the HTTP request (JSON body, URL parameters, headers, etc.).
2.  **Validate**: The populated struct is validated based on `validate` tags.
3.  **Execute**: The corresponding service function (business logic) is called, receiving the validated request struct as a parameter.
4.  **Respond**: The return value or error from the service function is automatically formatted into a standardized JSON response with the correct HTTP status code.

### 2. Decoupled Request Modals

Instead of binding directly to a `gin` context, you define request contracts as structs that implement the `RequestParam` interface, usually by embedding `api.RequestModal[T]`.

This approach, centered on the `RequestLoad` method, decouples your request definition from the web framework. Each request struct knows how to load itself from a generic `context.Context`, making the system adaptable to other frameworks in the future.

**Simple Request:**
```go
import "github.com/kod2ulz/gostart/api"

type MyRequest struct {
    Name string `json:"name" validate:"required"`
    api.RequestModal[MyRequest] // Embed to get default behavior
}
```

**Complex Request with Custom Loading:**
For complex scenarios, like reading from headers or multiple sources, you can override the `RequestLoad` method.

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/kod2ulz/gostart/api"
)

type TokenRequest struct {
    Token string `json:"token" validate:"required"`
    api.RequestModal[TokenRequest]
}

// Custom loading logic
func (r TokenRequest) RequestLoad(ctx context.Context) (param api.RequestParam, err error) {
    var out TokenRequest
    // Try to load token from "Authorization: Bearer <token>" header first
    if authHeader := ctx.(*gin.Context).Request.Header.Get("Authorization"); authHeader != "" {
        // ... parsing logic ...
        out.Token = parsedToken
        ctx.(*gin.Context).Set(out.ContextKey(), out)
        return out, nil
    }
    // Fallback to loading from JSON body
    if err = out.LoadFromJsonBody(ctx, &out); err != nil {
        return nil, err
    }
    ctx.(*gin.Context).Set(out.ContextKey(), out)
    return out, nil
}
```

### 3. Boilerplate-Free Handlers

Generic handlers connect a `gin` route to your service method in a single line. They infer the request and response types from your function's signature.

- `RequestHandlerWithResponse[RequestType, ResponseType]`: For single-item responses.
- `RequestHandlerWithListResponse[RequestType, ResponseType]`: For list/slice responses.

```go
// Your service method has a clean, framework-agnostic signature
func (s *service) CreateUser(ctx context.Context, params CreateUserRequest) (UserResponse, api.Error) {
    // ... business logic ...
}

// In your router setup:
router.POST("/users", api.RequestHandlerWithResponse(s.CreateUser))
```

### 4. Standardized Responses & Errors

The package provides `DataResponse`, `ListResponse`, and `ErrorResponse` wrappers to ensure all API outputs share a consistent JSON structure (`success`, `data`, `error`, `meta`, `references`, `time`). The custom `api.Error` interface and helpers (`ValidatorError`, `SQLError`, etc.) make returning detailed, structured errors simple.

## Roadmap

Future enhancements for this package include:

- **Enhanced Error Parsing**: Deeper inspection of database and validator errors to provide even more specific and helpful error messages (e.g., "user with this email already exists" from a SQL unique constraint violation).
- **Framework Agnosticism**: The long-term vision is to leverage the `RequestParam` interface to allow the request/response lifecycle to be used with other web frameworks like Fiber, Echo, or the standard `net/http` library.
- **Automated API Documentation**: Structure the API definitions in a way that enables the automatic generation of OpenAPI (Swagger) specifications.
