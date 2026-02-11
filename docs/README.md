# API Documentation Generation

This package provides automated OpenAPI/Swagger documentation generation for the GoStart framework. It uses reflection and struct tags to generate comprehensive API documentation without requiring manual annotations.

## Features

- **Zero-Annotation Documentation**: Automatically generates OpenAPI 3.0 specifications from your Go code
- **Tag-Driven Discovery**: Uses struct tags (`json:`, `query:`, `param:`, `header:`) to understand API structure
- **Type-Safe**: Leverages Go's type system to ensure accurate documentation
- **Framework Agnostic**: Works with any web framework through the contracts interface
- **Reflection-Based**: Automatically discovers routes and handlers
- **OpenAPI 3.0 Compliant**: Generates standard OpenAPI specifications

## Quick Start

### 1. Basic Usage

```go
package main

import (
    "github.com/kod2ulz/gostart/docs"
    "github.com/kod2ulz/gostart/contracts"
)

// Define your request type with tags
type UserRequest struct {
    ID    string `json:"id" param:"id" description:"User ID"`
    Name  string `json:"name" validate:"required" description:"User name"`
    Email string `json:"email" validate:"required,email" description:"User email"`
}

// Define your handler
func GetUserHandler(ctx contracts.RequestContext) {
    // Handler implementation
}

func main() {
    // Create documentation registry
    registry := docs.NewHandlerRegistry(&docs.DocumentationConfig{
        BaseURL:    "http://localhost:8080",
        Title:      "User API",
        Version:    "1.0.0",
        OutputFile: "swagger.json",
    })

    // Register your route
    registry.GET("/users/{id}", GetUserHandler).
        Description("Get user by ID").
        RequestType(reflect.TypeOf(UserRequest{}))

    // Generate documentation
    spec, err := registry.GenerateDocumentation()
    if err != nil {
        panic(err)
    }

    // Save or serve the documentation
    jsonData, _ := json.MarshalIndent(spec, "", "  ")
    fmt.Println(string(jsonData))
}
```

### 2. Integration with App Package

```go
package main

import (
    "github.com/kod2ulz/gostart/app"
    "github.com/kod2ulz/gostart/docs"
    "github.com/kod2ulz/gostart/contracts"
)

func main() {
    // Create documentation registry
    docRegistry := docs.NewHandlerRegistry(&docs.DocumentationConfig{
        BaseURL: "http://localhost:8080",
        Title:   "My Application API",
        Version: "1.0.0",
    })

    // Initialize app
    application := app.Init()

    // Register routes and document them
    application.Router().GET("/users", func(ctx contracts.RequestContext) {
        // Handler implementation
    })

    // Register for documentation
    docRegistry.GET("/users", userHandler).
        Description("Get all users").
        RequestType(reflect.TypeOf(UserRequest{}))

    // Add documentation endpoint
    application.Router().GET("/swagger.json", func(ctx contracts.RequestContext) {
        spec, _ := docRegistry.GenerateDocumentation()
        ctx.JSON(200, spec)
    })

    // Run the application
    application.Run()
}
```

## Struct Tags

The documentation system uses standard struct tags to understand your API:

### Request Source Tags

- `json:"field"` - Field comes from request body JSON
- `query:"param"` - Field comes from URL query parameter
- `param:"name"` - Field comes from URL path parameter
- `header:"X-Header"` - Field comes from HTTP header

### Documentation Tags

- `description:"text"` - Human-readable field description
- `validate:"required"` - Indicates required field
- `example:"value"` - Example value for the field

### Example

```go
type CreateUserRequest struct {
    Username string `json:"username" validate:"required" description:"Unique username"`
    Email    string `json:"email" validate:"required,email" description:"User email address"`
    Age      int    `json:"age" query:"age" description:"User age (optional)"`
    APIKey   string `json:"-" header:"X-API-Key" description:"API key for authentication"`
}
```

## Advanced Usage

### Custom Documentation Configuration

```go
config := &docs.DocumentationConfig{
    BaseURL:    "https://api.example.com/v1",
    Title:      "My API Documentation",
    Version:    "1.0.0",
    OutputFile: "api-docs.json",
    Format:     "json", // or "yaml"
}
```

### Route Builder Pattern

```go
registry.POST("/users", CreateUserHandler).
    Description("Create a new user").
    RequestType(reflect.TypeOf(CreateUserRequest{})).
    HandlerFunc(CreateUserHandler)
```

### HTTP Documentation Endpoint

```go
// Serve documentation via HTTP
app.Router().GET("/docs", func(ctx contracts.RequestContext) {
    spec, _ := docRegistry.GenerateDocumentation()
    ctx.JSON(200, spec)
})
```

## How It Works

1. **Type Analysis**: Uses reflection to analyze your request/response types
2. **Tag Parsing**: Reads struct tags to understand field sources and validation
3. **Schema Generation**: Creates JSON Schema definitions for your types
4. **OpenAPI Generation**: Builds complete OpenAPI 3.0 specifications
5. **Format Output**: Outputs JSON or YAML documentation

## Integration Points

### With Router Abstraction

The documentation system integrates seamlessly with the framework-agnostic router:

```go
// Routes registered with the router can be automatically documented
router.GET("/users", handler)
docRegistry.GET("/users", handler).Description("Get users")
```

### With Tag-Driven Loading

Works perfectly with the automatic request loading system:

```go
// The same tags used for loading are used for documentation
type Request struct {
    ID   string `param:"id"`      // Loaded from path, documented as path parameter
    Name string `json:"name"`     // Loaded from body, documented as body field
    Auth string `header:"Auth"`   // Loaded from header, documented as header
}
```

## Examples

See `example_usage.go` for complete examples of:

- Basic documentation generation
- Complex type handling
- Integration patterns
- Advanced configurations

## Benefits

- **Always Up-to-Date**: Documentation is generated from actual code
- **No Manual Effort**: No need to write or maintain annotations
- **Type Safety**: Documentation matches your actual types
- **Consistent**: Standardized documentation across all endpoints
- **Developer Friendly**: Automatic discovery and generation

## Future Enhancements

- YAML output support
- Interactive documentation UI (Swagger UI integration)
- Advanced comment parsing
- Automated testing integration
- Client code generation
- Versioned documentation