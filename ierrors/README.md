# IErrors Package

The `ierrors` package provides the foundational error interface and types for the GoStart framework. It serves as the base layer for all error handling throughout the application, providing a unified interface for error management.

## Overview

The `ierrors` package defines the core `Error` interface that all errors in the GoStart ecosystem implement. This interface provides chainable methods for adding context, HTTP status codes, and error metadata.

## Core Interface

### Error Interface

```go
type Error interface {
    error
    WithMessage(msg string) Error
    WithMessagef(format string, args ...interface{}) Error
    WithErrorCode(code string) Error
    WithHttpStatusCode(code int) Error
    WithContext(key string, value interface{}) Error
    WithContextValue(key string, value interface{}) Error
    WithContextValues(values map[string]interface{}) Error
    WithCause(err error) Error
    WithUserMessage(msg string) Error
    WithUserMessagef(format string, args ...interface{}) Error
    Unwrap() error
    GetMessage() string
    GetErrorCode() string
    GetHttpStatusCode() int
    GetContext() map[string]interface{}
    GetUserMessage() string
    GetTimestamp() time.Time
    Is(target error) bool
}
```

## Usage Examples

### Basic Error Creation

```go
import "github.com/kod2ulz/gostart/ierrors"

// Create a basic error
err := ierrors.Errorf("something went wrong")

// Add context to the error
contextErr := err.
    WithErrorCode("PROCESSING_ERROR").
    WithHttpStatusCode(http.StatusInternalServerError).
    WithContext("operation", "user_creation").
    WithContext("user_id", "123")

// Check error type
if ierrors.Is(err, ierrors.Errorf("something went wrong")) {
    // Handle specific error
}
```

### Error Wrapping

```go
// Wrap an existing error with additional context
originalErr := database.Exec("INSERT INTO users...")
enhancedErr := ierrors.Wrap(originalErr, "failed to create user").
    WithErrorCode("DATABASE_ERROR").
    WithHttpStatusCode(http.StatusServiceUnavailable).
    WithContext("table", "users").
    WithUserMessage("Unable to create user at this time. Please try again later.")
```

### Error Comparison

```go
// Check if error matches a specific type
var notFoundErr *NotFoundError
if errors.As(err, &notFoundErr) {
    // Handle not found error
}

// Check if error contains specific context
if err.GetErrorCode() == "VALIDATION_ERROR" {
    // Handle validation error
}

// Check HTTP status code
if err.GetHttpStatusCode() == http.StatusNotFound {
    // Handle 404 error
}
```

### Error Context

```go
// Add multiple context values
err := ierrors.Errorf("validation failed").
    WithContextValues(map[string]interface{}{
        "field":     "email",
        "value":     "invalid-email",
        "required":  true,
        "rule":      "email_format",
    })

// Retrieve context values
if ctx := err.GetContext(); ctx != nil {
    if field, ok := ctx["field"]; ok {
        fmt.Printf("Error field: %v\n", field)
    }
}
```

### User-Friendly Messages

```go
// Technical error with user-friendly message
err := ierrors.Errorf("database constraint violation: unique_email").
    WithErrorCode("DUPLICATE_ENTRY").
    WithHttpStatusCode(http.StatusConflict).
    WithUserMessage("A user with this email address already exists.")

// In handlers, return appropriate message based on audience
func handleError(err ierrors.Error) {
    if err.GetUserMessage() != "" {
        // Return user-friendly message to client
        return err.GetUserMessage()
    }

    // Return technical message for internal logging
    return err.GetMessage()
}
```

### HTTP Status Code Management

```go
// Create error with appropriate HTTP status
err := ierrors.Errorf("user not found").
    WithErrorCode("NOT_FOUND").
    WithHttpStatusCode(http.StatusNotFound)

// Common HTTP status codes
statusCodes := map[string]int{
    "BAD_REQUEST":           http.StatusBadRequest,
    "UNAUTHORIZED":          http.StatusUnauthorized,
    "FORBIDDEN":             http.StatusForbidden,
    "NOT_FOUND":             http.StatusNotFound,
    "CONFLICT":              http.StatusConflict,
    "UNPROCESSABLE_ENTITY":  http.StatusUnprocessableEntity,
    "TOO_MANY_REQUESTS":     http.StatusTooManyRequests,
    "INTERNAL_SERVER_ERROR": http.StatusInternalServerError,
    "SERVICE_UNAVAILABLE":   http.StatusServiceUnavailable,
}
```

## Error Chain Patterns

### Error Chaining

```go
// Chain multiple operations with error context
func ProcessUser(user User) error {
    if err := validateUser(user); err != nil {
        return ierrors.Wrap(err, "user validation failed").
            WithErrorCode("VALIDATION_ERROR").
            WithHttpStatusCode(http.StatusBadRequest)
    }

    if err := saveUser(user); err != nil {
        return ierrors.Wrap(err, "failed to save user").
            WithErrorCode("DATABASE_ERROR").
            WithHttpStatusCode(http.StatusInternalServerError)
    }

    return nil
}
```

### Error Aggregation

```go
// Collect multiple errors
func BatchProcess(items []Item) ierrors.Error {
    var errs []ierrors.Error

    for _, item := range items {
        if err := processItem(item); err != nil {
            errs = append(errs, ierrors.Wrap(err, fmt.Sprintf("failed to process item %s", item.ID)))
        }
    }

    if len(errs) > 0 {
        return ierrors.Errorf("batch processing failed").
            WithErrorCode("BATCH_ERROR").
            WithContext("failed_items", len(errs)).
            WithContext("total_items", len(items)).
            WithCause(ierrors.Join(errs...))
    }

    return nil
}
```

## Best Practices

### 1. Error Code Conventions

```go
// Use consistent error code naming
const (
    ErrorCodeValidation      = "VALIDATION_ERROR"
    ErrorCodeAuthentication = "AUTHENTICATION_ERROR"
    ErrorCodeAuthorization  = "AUTHORIZATION_ERROR"
    ErrorCodeNotFound       = "NOT_FOUND"
    ErrorCodeConflict       = "CONFLICT"
    ErrorCodeInternal       = "INTERNAL_ERROR"
)
```

### 2. HTTP Status Code Mapping

```go
// Map error codes to HTTP status codes
func GetHttpStatusCode(err ierrors.Error) int {
    switch err.GetErrorCode() {
    case ErrorCodeValidation:
        return http.StatusBadRequest
    case ErrorCodeAuthentication:
        return http.StatusUnauthorized
    case ErrorCodeAuthorization:
        return http.StatusForbidden
    case ErrorCodeNotFound:
        return http.StatusNotFound
    case ErrorCodeConflict:
        return http.StatusConflict
    default:
        return http.StatusInternalServerError
    }
}
```

### 3. Error Logging

```go
// Log errors with full context
func logError(err ierrors.Error) {
    log.Printf("Error: %s\n", err.GetMessage())
    log.Printf("Code: %s\n", err.GetErrorCode())
    log.Printf("HTTP Status: %d\n", err.GetHttpStatusCode())
    log.Printf("Context: %v\n", err.GetContext())
    log.Printf("User Message: %s\n", err.GetUserMessage())

    // Log stack trace if available
    if cause := errors.Unwrap(err); cause != nil {
        log.Printf("Cause: %v\n", cause)
    }
}
```

### 4. Error Handling in Handlers

```go
// Handle errors in API handlers
func UserHandler(ctx contracts.RequestContext) (User, ierrors.Error) {
    var req GetUserRequest

    if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
        return User{}, err
    }

    user, err := userService.GetUser(req.ID)
    if err != nil {
        // Error already has proper HTTP status code and context
        return User{}, err
    }

    return user, nil
}
```

## Integration with Higher-Level Packages

The `ierrors` package is designed to work seamlessly with the `errors` package, which provides enhanced error parsing and specific error types:

```go
// Use with errors package for enhanced error handling
import "github.com/kod2ulz/gostart/errors"

func ServiceMethod() ierrors.Error {
    // Validation errors
    if err := validate(); err != nil {
        return errors.ValidationFailed[MyType](err)
    }

    // Database errors
    if err := databaseOperation(); err != nil {
        return errors.SQLError[MyType](err)
    }

    // Not found errors
    if item == nil {
        return errors.NotFound[MyType]("item not found")
    }

    return nil
}
```

## Performance Considerations

1. **Error Creation**: Error creation is relatively expensive. Avoid creating errors in hot paths.
2. **Context Storage**: Context values are stored in memory. Be mindful of what you store in error context.
3. **Error Chaining**: Deep error chains can impact performance. Keep error chains reasonable.

## Migration from Standard Errors

```go
// Before: standard Go errors
if err != nil {
    return fmt.Errorf("failed to process: %w", err)
}

// After: ierrors with enhanced context
if err != nil {
    return ierrors.Wrap(err, "failed to process").
        WithErrorCode("PROCESSING_ERROR").
        WithHttpStatusCode(http.StatusInternalServerError)
}
```

The `ierrors` package provides the foundation for robust error handling throughout the GoStart framework, enabling consistent error management with rich context and user-friendly messaging.