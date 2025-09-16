# Error Handling Package

The `errors` package provides comprehensive error handling and parsing utilities for GoStart applications. It offers intelligent error message generation, validation error parsing, and database error transformation to provide user-friendly error responses.

## Features

- **Enhanced Error Parsing**: Automatic parsing of validation errors, SQL errors, and common error patterns
- **User-Friendly Messages**: Transforms technical error messages into actionable feedback for frontend consumption
- **Centralized Error Handling**: All error processing logic centralized in one package
- **Multi-Database Support**: Specialized parsing for PostgreSQL, MySQL, and SQLite errors
- **Validation Error Enhancement**: Detailed field-level validation errors with rule extraction
- **Framework Agnostic**: Works with any web framework through the unified error interface

## Error Types

### Validation Errors

Validation errors are automatically parsed from go-playground/validator errors and provide detailed field-level information.

```go
import "github.com/kod2ulz/gostart/errors"

// In your service layer
func CreateUser(user CreateUserRequest) (User, error) {
    if err := validator.Struct(user); err != nil {
        return User{}, errors.ValidationFailed[CreateUserRequest](err)
    }
    // ... create user logic
}
```

**Example Output - Single Field Validation Error:**
```json
{
  "success": false,
  "type": "validation_error",
  "time": 1634567890,
  "error": {
    "code": "ValidationError",
    "message": "email is required",
    "fields": {
      "email": "email is required"
    }
  }
}
```

**Example Output - Multiple Field Validation Errors:**
```json
{
  "success": false,
  "type": "validation_error",
  "time": 1634567890,
  "error": {
    "code": "ValidationError",
    "message": "Validation failed for 3 fields",
    "fields": {
      "email": "email must be a valid email address",
      "password": "password must be at least 8 characters",
      "age": "age must be greater than or equal to 18"
    }
  }
}
```

### SQL Errors

Database errors are automatically parsed and transformed into user-friendly messages.

```go
import "github.com/kod2ulz/gostart/errors"

// In your data access layer
func CreateUser(user User) error {
    _, err := db.ExecContext(ctx,
        "INSERT INTO users (email, name) VALUES ($1, $2)",
        user.Email, user.Name)

    if err != nil {
        return errors.SQLError[User](err)
    }
    return nil
}
```

#### PostgreSQL Errors

**Duplicate Entry Error:**
```json
{
  "success": false,
  "type": "error",
  "time": 1634567890,
  "error": {
    "code": "DUPLICATE_ENTRY",
    "message": "A record with this Email already exists",
    "fields": {
      "email": "A record with this Email already exists"
    }
  }
}
```

**Foreign Key Violation:**
```json
{
  "success": false,
  "type": "error",
  "time": 1634567890,
  "error": {
    "code": "FOREIGN_KEY_VIOLATION",
    "message": "Referenced Department does not exist",
    "fields": {
      "department_id": "Referenced Department does not exist"
    }
  }
}
```

**Not Null Violation:**
```json
{
  "success": false,
  "type": "error",
  "time": 1634567890,
  "error": {
    "code": "REQUIRED_FIELD",
    "message": "Email is required",
    "fields": {
      "email": "Email is required"
    }
  }
}
```

**String Truncation Error:**
```json
{
  "success": false,
  "type": "error",
  "time": 1634567890,
  "error": {
    "code": "VALUE_TOO_LONG",
    "message": "Username exceeds maximum allowed length",
    "fields": {
      "username": "Username exceeds maximum allowed length"
    }
  }
}
```

#### MySQL Errors

**Duplicate Entry Error:**
```json
{
  "success": false,
  "type": "error",
  "time": 1634567890,
  "error": {
    "code": "DUPLICATE_ENTRY",
    "message": "A record with these values already exists",
    "details": {
      "duplicate_value": "john@example.com",
      "key_name": "users_email_unique"
    }
  }
}
```

#### SQLite Errors

**Unique Constraint Error:**
```json
{
  "success": false,
  "type": "error",
  "time": 1634567890,
  "error": {
    "code": "DUPLICATE_ENTRY",
    "message": "A record with this username already exists",
    "fields": {
      "username": "A record with this username already exists"
    }
  }
}
```

### Request Loading Errors

Errors that occur during request parameter binding and validation.

```go
import "github.com/kod2ulz/gostart/errors"

// Automatic error handling in handlers
func HandleUserRequest(ctx contracts.RequestContext) {
    param, err := LoadRequestParam(ctx)
    if err != nil {
        // This automatically uses RequestLoadError with enhanced parsing
        HandleError(ctx, errors.RequestLoadError[UserRequest](err))
        return
    }
    // ... process request
}
```

**JSON Binding Error:**
```json
{
  "success": false,
  "type": "error",
  "time": 1634567890,
  "error": {
    "code": "RequestLoadError",
    "message": "Invalid JSON format in request body"
  }
}
```

**Required Field Missing:**
```json
{
  "success": false,
  "type": "error",
  "time": 1634567890,
  "error": {
    "code": "RequestLoadError",
    "message": "Required fields are missing from request"
  }
}
```

### General Server Errors

For unexpected errors and server issues.

```go
import "github.com/kod2ulz/gostart/errors"

// In your service layer
func ProcessComplexOperation(data Data) error {
    result, err := externalService.Call(data)
    if err != nil {
        return errors.ServerError(err)
    }
    return nil
}
```

**Server Error Output:**
```json
{
  "success": false,
  "type": "error",
  "time": 1634567890,
  "error": {
    "code": "ServerError",
    "message": "Internal server error"
  }
}
```

### Not Found Errors

For when a requested resource is not found.

```go
import "github.com/kod2ulz/gostart/errors"

// In your data access layer
func GetUserByID(id int) (User, error) {
    user, err := db.GetUserByID(id)
    if err != nil {
        return User{}, errors.SqlQueryError[id, User](id, User{}, err)
    }
    return user, nil
}
```

**Not Found Error Output:**
```json
{
  "success": false,
  "type": "error",
  "time": 1634567890,
  "error": {
    "code": "NotFoundError",
    "message": "Not Found",
    "params": 123
  }
}
```

## Usage Examples

### Basic Error Handling

```go
package main

import (
    "github.com/kod2ulz/gostart/api"
    "github.com/kod2ulz/gostart/errors"
)

type CreateUserRequest struct {
    api.RequestModal[CreateUserRequest]
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Name     string `json:"name" validate:"required"`
}

func CreateUserHandler(ctx contracts.RequestContext) {
    var req CreateUserRequest
    if err := req.RequestLoad(ctx); err != nil {
        errors.HandleError(ctx, errors.RequestLoadError[CreateUserRequest](err))
        return
    }

    // Validate request
    if err := req.Validate(ctx); err != nil {
        errors.HandleError(ctx, errors.ValidationFailed[CreateUserRequest](err))
        return
    }

    // Create user in database
    if err := userRepository.Create(req); err != nil {
        errors.HandleError(ctx, errors.SQLError[CreateUserRequest](err))
        return
    }

    api.SuccessResponse(ctx, "User created successfully")
}
```

### Custom Error Handling

```go
package main

import (
    "github.com/kod2ulz/gostart/errors"
    "github.com/kod2ulz/gostart/ierrors"
)

// Custom error handling logic
func HandleCustomError(ctx contracts.RequestContext, err ierrors.Error) {
    // Use centralized error handling
    errorCode, errorMessage, httpCode, fields := errors.HandleAPIError(err)

    // Add custom logic based on error type
    switch errorCode {
    case "ValidationError":
        api.ValidationError(ctx, errorMessage, fields)
    case "DUPLICATE_ENTRY":
        api.ErrorResponse(ctx, errorCode, errorMessage, http.StatusConflict)
    default:
        api.ErrorResponse(ctx, errorCode, errorMessage, httpCode)
    }
}
```

### Service Layer Error Handling

```go
package service

import (
    "github.com/kod2ulz/gostart/errors"
)

type UserService struct {
    repo UserRepository
}

func (s *UserService) CreateUser(user User) (User, error) {
    // Business logic validation
    if !s.isValidEmail(user.Email) {
        return User{}, errors.ValidationFailed[User](fmt.Errorf("invalid email format"))
    }

    // Database operation
    result, err := s.repo.Create(user)
    if err != nil {
        return User{}, errors.SQLError[User](err)
    }

    return result, nil
}

func (s *UserService) GetUserByID(id int) (User, error) {
    user, err := s.repo.FindByID(id)
    if err != nil {
        return User{}, errors.SqlQueryError[id, User](id, User{}, err)
    }
    return user, nil
}
```

## Error Helper Functions

### Validation Errors

- `ValidationFailed[T any](err error)` - Enhanced validation error parsing for go-playground/validator
- `IsValidationError(err ierrors.Error)` - Check if error is a validation error
- `HandleValidationError(err ierrors.Error)` - Process validation error and return structured information

### SQL Errors

- `SQLError[T any](err error)` - Enhanced SQL error with detailed parsing
- `ParseSQLError(err error)` - Parse database errors and extract meaningful information
- `SqlQueryError[P any, T any](param P, out T, err error)` - Handle SQL query errors with automatic not-found detection
- `SqlNoRows(err error)` - Check if error is a "no rows" error

### General Errors

- `GeneralError[T any](err error)` - Create a general error with enhanced message parsing
- `ServerError(err error)` - Create a server error
- `NotFoundError[T any, P any](param P)` - Create a not found error
- `RequestLoadError[T any](err error)` - Enhanced request loading error parsing

### Centralized Error Handling

- `HandleAPIError(err ierrors.Error)` - Centralized error handling for API responses

## HTTP Status Code Mapping

The errors package automatically maps error types to appropriate HTTP status codes:

- `ValidationError`: 400 Bad Request
- `RequestLoadError`: 400 Bad Request
- `NotFoundError`: 404 Not Found
- `Unauthorized`: 401 Unauthorized
- `DUPLICATE_ENTRY`, `FOREIGN_KEY_VIOLATION`, `CONSTRAINT_VIOLATION`: 409 Conflict
- `REQUIRED_FIELD`, `VALUE_TOO_LONG`, `VALUE_OUT_OF_RANGE`: 400 Bad Request
- `ServerError`, `SQLError`: 500 Internal Server Error

## Best Practices

1. **Use Type-Specific Error Functions**: Always use the most specific error function for your error type (e.g., `ValidationFailed` for validation errors, `SQLError` for database errors).

2. **Centralize Error Handling**: Use the centralized `HandleAPIError` function in your handlers to ensure consistent error responses.

3. **Provide Context**: When possible, provide context parameters to error functions (e.g., user ID in `NotFoundError`).

4. **Handle Errors at the Appropriate Level**:
   - Use validation errors in the service layer
   - Use SQL errors in the data access layer
   - Use centralized error handling in the API handlers

5. **Test Error Scenarios**: Test various error scenarios to ensure proper error messages and HTTP status codes are returned.

## Migration from Previous Versions

If you're upgrading from a previous version, the main changes are:

1. **Centralized Error Handling**: All error parsing logic is now centralized in the `errors` package
2. **Enhanced Error Messages**: Error messages are now more user-friendly and actionable
3. **Automatic Field Extraction**: Database and validation errors automatically extract field-level information
4. **New Validation Function**: Use `ValidationFailed[T any]()` instead of `ValidatorError[T any]()`

```go
// Old way
return errors.ValidatorError[User](err)

// New way
return errors.ValidationFailed[User](err)
```