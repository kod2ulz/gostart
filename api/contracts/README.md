# Contract Analysis System

The Contract Analysis System provides intelligent detection and analysis of API contracts, automatically extracting request/response specifications from handler functions and generating comprehensive OpenAPI documentation.

## Overview

This system analyzes your Go code to automatically determine:

- **Request Contracts**: Path parameters, query strings, headers, cookies, and request body schemas
- **Response Contracts**: Response body schemas, headers, error responses, and content types
- **Type Schemas**: JSON schemas generated from Go structs with validation tags
- **Error Handling**: Automatic detection and documentation of error responses

## Features

### 1. Automatic Request Contract Detection

The system analyzes handler functions to extract request information:

```go
// Example route: GET /users/:id
func getUserHandler(ctx contracts.RequestContext) {
    // Automatically detects:
    // - Path parameter: id (string, required)
    // - Query parameters: filter, limit, offset
    // - Headers: Authorization, Content-Type
    // - Cookies: session
}
```

**Detected Request Contract:**
```json
{
  "pathParameters": {
    "id": {
      "name": "id",
      "type": "string",
      "required": true,
      "in": "path"
    }
  },
  "queryParameters": {
    "filter": {
      "name": "filter",
      "type": "string",
      "required": false,
      "in": "query"
    },
    "limit": {
      "name": "limit",
      "type": "integer",
      "required": false,
      "in": "query"
    },
    "offset": {
      "name": "offset",
      "type": "integer",
      "required": false,
      "in": "query"
    }
  },
  "headers": {
    "Authorization": {
      "name": "Authorization",
      "type": "string",
      "required": false,
      "in": "header"
    },
    "Content-Type": {
      "name": "Content-Type",
      "type": "string",
      "required": false,
      "in": "header",
      "default": "application/json"
    }
  },
  "cookies": {
    "session": {
      "name": "session",
      "type": "string",
      "required": false,
      "in": "cookie"
    }
  },
  "body": {
    "contentType": "application/json",
    "required": true,
    "schema": {
      "type": "object",
      "properties": {
        "data": {
          "type": "object"
        }
      }
    }
  }
}
```

### 2. Response Contract Detection

The system analyzes return types to determine response structure:

```go
func createUserHandler(ctx contracts.RequestContext) (User, error) {
    // Automatically detects:
    // - 200 response with User schema
    // - Error responses (400, 401, 404, 500)
    // - Response headers
}
```

**Detected Response Contract:**
```json
{
  "statusCode": 200,
  "contentType": "application/json",
  "headers": {
    "Content-Type": {
      "name": "Content-Type",
      "type": "string",
      "required": true,
      "in": "header",
      "default": "application/json"
    },
    "X-Request-ID": {
      "name": "X-Request-ID",
      "type": "string",
      "required": false,
      "in": "header",
      "description": "Unique request identifier"
    }
  },
  "body": {
    "contentType": "application/json",
    "required": true,
    "schema": {
      "type": "object",
      "properties": {
        "success": {
          "type": "boolean",
          "default": true
        },
        "data": {
          "$ref": "#/components/schemas/User"
        }
      }
    }
  },
  "errorResponses": {
    "400": {
      "statusCode": 400,
      "contentType": "application/json",
      "body": {
        "contentType": "application/json",
        "schema": {
          "type": "object",
          "properties": {
            "success": {
              "type": "boolean",
              "default": false
            },
            "error": {
              "type": "object",
              "properties": {
                "code": {
                  "type": "string"
                },
                "message": {
                  "type": "string"
                },
                "fields": {
                  "type": "object"
                }
              }
            }
          }
        }
      }
    },
    "401": {
      "statusCode": 401,
      "contentType": "application/json",
      "body": {
        "contentType": "application/json",
        "schema": {
          "type": "object",
          "properties": {
            "success": {
              "type": "boolean",
              "default": false
            },
            "error": {
              "type": "object",
              "properties": {
                "code": {
                  "type": "string"
                },
                "message": {
                  "type": "string"
                }
              }
            }
          }
        }
      }
    }
  }
}
```

### 3. Type Schema Generation

The system automatically converts Go structs to JSON schemas:

```go
type User struct {
    ID       string `json:"id" example:"123"`
    Name     string `json:"name" validate:"required" example:"John Doe"`
    Email    string `json:"email" validate:"required,email" example:"john@example.com"`
    Age      int    `json:"age" validate:"gte=18,lte=120" example:"30"`
    Active   bool   `json:"active" default:"true"`
    CreatedAt string `json:"createdAt" readonly:"true"`
}
```

**Generated Schema:**
```json
{
  "type": "object",
  "properties": {
    "id": {
      "type": "string",
      "example": "123"
    },
    "name": {
      "type": "string",
      "minLength": 1,
      "example": "John Doe"
    },
    "email": {
      "type": "string",
      "format": "email",
      "example": "john@example.com"
    },
    "age": {
      "type": "integer",
      "minimum": 18,
      "maximum": 120,
      "example": 30
    },
    "active": {
      "type": "boolean",
      "default": true
    },
    "createdAt": {
      "type": "string",
      "readOnly": true
    }
  },
  "required": ["name", "email"]
}
```

### 4. Validation Tag Support

The system supports common validation tags:

```go
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=2,max=100"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"gte=18,lte=120"`

    // Special formats
    URL    string `json:"url" validate:"url"`
    UUID   string `json:"uuid" validate:"uuid"`

    // String validation
    Bio    string `json:"bio" validate:"max=500"`

    // Array validation
    Tags   []string `json:"tags" validate:"max=10,dive,required"`
}
```

**Generated Schema with Validation:**
```json
{
  "type": "object",
  "properties": {
    "name": {
      "type": "string",
      "minLength": 2,
      "maxLength": 100
    },
    "email": {
      "type": "string",
      "format": "email"
    },
    "age": {
      "type": "integer",
      "minimum": 18,
      "maximum": 120
    },
    "url": {
      "type": "string",
      "format": "uri"
    },
    "uuid": {
      "type": "string",
      "format": "uuid"
    },
    "bio": {
      "type": "string",
      "maxLength": 500
    },
    "tags": {
      "type": "array",
      "maxItems": 10,
      "items": {
        "type": "string",
        "minLength": 1
      }
    }
  },
  "required": ["name", "email"]
}
```

## Usage

### 1. Direct Contract Analysis

```go
package main

import (
    "fmt"
    "log"

    "github.com/kod2ulz/gostart/api/contracts"
)

type User struct {
    ID    string `json:"id"`
    Name  string `json:"name" validate:"required"`
    Email string `json:"email" validate:"required,email"`
}

func main() {
    // Create contract analyzer
    analyzer := contracts.NewContractAnalyzer()

    // Analyze a handler function
    handler := func(ctx contracts.RequestContext) (*User, error) {
        return &User{
            ID:    "123",
            Name:  "John Doe",
            Email: "john@example.com",
        }, nil
    }

    // Analyze request contract
    requestContract, err := analyzer.AnalyzeRequest(handler, "/users/:id")
    if err != nil {
        log.Fatal(err)
    }

    // Analyze response contract
    responseContract, err := analyzer.AnalyzeResponse(handler)
    if err != nil {
        log.Fatal(err)
    }

    // Print contracts
    fmt.Printf("Request Contract: %+v\n", requestContract)
    fmt.Printf("Response Contract: %+v\n", responseContract)
}
```

### 2. Type Schema Analysis

```go
func main() {
    analyzer := contracts.NewContractAnalyzer()

    // Analyze a struct type
    userType := reflect.TypeOf(User{})
    schema := analyzer.AnalyzeType(userType)

    // Convert to JSON
    jsonData, _ := json.MarshalIndent(schema, "", "  ")
    fmt.Println("Generated Schema:")
    fmt.Println(string(jsonData))
}
```

### 3. Custom Type Mappings

You can add custom type mappings for special handling:

```go
func main() {
    analyzer := contracts.NewContractAnalyzer()

    // Add custom mapping for time.Time
    analyzer.customMappings["time.Time"] = func(t reflect.Type) *contracts.SchemaContract {
        return &contracts.SchemaContract{
            Type:   "string",
            Format: "date-time",
        }
    }

    // Add custom mapping for uuid.UUID
    analyzer.customMappings["uuid.UUID"] = func(t reflect.Type) *contracts.SchemaContract {
        return &contracts.SchemaContract{
            Type:   "string",
            Format: "uuid",
        }
    }

    // Use the analyzer
    schema := analyzer.AnalyzeType(reflect.TypeOf(time.Time{}))
    fmt.Printf("Time schema: %+v\n", schema)
}
```

## Advanced Features

### 1. Custom Request Body Detection

The system can detect different request body types:

```go
func createUserHandler(ctx contracts.RequestContext) error {
    // JSON body detection
    var user User
    if err := ctx.ShouldBindJSON(&user); err != nil {
        return err
    }

    // XML body detection
    var xmlUser User
    if err := ctx.ShouldBindXML(&xmlUser); err != nil {
        return err
    }

    // Form data detection
    var formUser User
    if err := ctx.ShouldBind(&formUser); err != nil {
        return err
    }

    return nil
}
```

### 2. Response Type Analysis

The system analyzes different response types:

```go
// Simple response
func simpleHandler() string {
    return "Hello World"
}

// Struct response
func userHandler() (*User, error) {
    return &User{ID: "123", Name: "John"}, nil
}

// Array response
func listUsersHandler() ([]User, error) {
    return []User{user1, user2}, nil
}

// Map response
func statusHandler() map[string]interface{} {
    return map[string]interface{}{
        "status": "healthy",
        "version": "1.0.0",
    }
}
```

### 3. Error Response Detection

The system automatically detects error handling patterns:

```go
func getUserHandler(ctx contracts.RequestContext) (*User, error) {
    id := ctx.Param("id")

    user, err := userService.GetUser(id)
    if err != nil {
        if errors.Is(err, ErrUserNotFound) {
            return nil, &UserError{
                Code:    "USER_NOT_FOUND",
                Message: "User not found",
                Status:  404,
            }
        }
        return nil, err
    }

    return user, nil
}
```

**Generated Error Responses:**
```json
{
  "400": {
    "description": "Bad request",
    "content": {
      "application/json": {
        "schema": {
          "type": "object",
          "properties": {
            "success": { "type": "boolean", "default": false },
            "error": {
              "type": "object",
              "properties": {
                "code": { "type": "string" },
                "message": { "type": "string" },
                "fields": { "type": "object" }
              }
            }
          }
        }
      }
    }
  },
  "404": {
    "description": "User not found",
    "content": {
      "application/json": {
        "schema": {
          "type": "object",
          "properties": {
            "success": { "type": "boolean", "default": false },
            "error": {
              "type": "object",
              "properties": {
                "code": { "type": "string" },
                "message": { "type": "string" }
              }
            }
          }
        }
      }
    }
  }
}
```

## Configuration

### 1. Analyzer Configuration

```go
analyzer := contracts.NewContractAnalyzer()

// Add known types
analyzer.knownTypes["User"] = reflect.TypeOf(User{})

// Add custom mappings
analyzer.customMappings["custom.Time"] = func(t reflect.Type) *contracts.SchemaContract {
    return &contracts.SchemaContract{
        Type:   "string",
        Format: "custom-time",
    }
}
```

### 2. Contract Customization

```go
// Customize request contract detection
requestContract, _ := analyzer.AnalyzeRequest(handler, "/users/:id")

// Add custom parameters
requestContract.QueryParameters["custom"] = contracts.ParameterContract{
    Name:        "custom",
    Type:        "string",
    Required:    false,
    Description: "Custom parameter",
    In:          "query",
}

// Customize response contract
responseContract, _ := analyzer.AnalyzeResponse(handler)

// Add custom headers
responseContract.Headers["X-Custom"] = contracts.ParameterContract{
    Name:        "X-Custom",
    Type:        "string",
    Required:    false,
    Description: "Custom header",
    In:          "header",
}
```

## Integration with OpenAPI

The contract analysis system integrates seamlessly with the OpenAPI generation system:

```go
// The OpenAPI generator automatically uses contract analysis
func (g *Generator) AddRoute(method, path string, handler interface{}, annotations map[string]interface{}) {
    // Analyze contracts
    requestContract, _ := g.contractAnalyzer.AnalyzeRequest(handler, path)
    responseContract, _ := g.contractAnalyzer.AnalyzeResponse(handler)

    // Store contracts in route annotations
    route.Annotations["requestContract"] = requestContract
    route.Annotations["responseContract"] = responseContract

    // Generate OpenAPI operation with enriched contract information
    operation := g.createOperation(route)
    g.enrichOperationFromRequestContract(operation, requestContract)
    g.enrichOperationFromResponseContract(operation, responseContract)
}
```

## Best Practices

### 1. Use Consistent Handler Signatures

```go
// Good: Clear return types
func getUserHandler(ctx contracts.RequestContext) (*User, error)

// Good: Structured response
func listUsersHandler(ctx contracts.RequestContext) ([]User, *int64, error)

// Avoid: Unclear return types
func getUserHandler(ctx contracts.RequestContext) interface{}
```

### 2. Add Struct Tags

```go
type User struct {
    ID       string `json:"id" example:"123" validate:"required"`
    Name     string `json:"name" validate:"required,min=2,max=100"`
    Email    string `json:"email" validate:"required,email"`
    Age      int    `json:"age" validate:"gte=18,lte=120"`
    Active   bool   `json:"active" default:"true"`
    CreatedAt time.Time `json:"createdAt" readonly:"true"`
}
```

### 3. Use Proper Error Types

```go
type APIError struct {
    Code    string                 `json:"code"`
    Message string                 `json:"message"`
    Fields  map[string]interface{} `json:"fields,omitempty"`
}

func (e *APIError) Error() string {
    return e.Message
}
```

## Troubleshooting

### Common Issues

1. **Missing Request Parameters**
   - Ensure route paths use proper parameter format (`:id` not `{id}`)
   - Check that handler functions use proper RequestContext methods

2. **Incorrect Response Types**
   - Verify handler return types are properly defined
   - Check that struct tags are correctly formatted

3. **Missing Validation Rules**
   - Ensure validation tags use correct syntax
   - Check that required fields are properly marked

4. **Circular References**
   - Avoid circular references in struct definitions
   - Use pointer types for recursive structures

## Performance Considerations

The contract analysis system is designed to be efficient:

- **Type Caching**: Analyzed types are cached to avoid re-analysis
- **Lazy Evaluation**: Contracts are only analyzed when needed
- **Minimal Reflection**: Reflection is used sparingly and cached
- **Concurrent Safe**: The analyzer is safe for concurrent use

## Integration with Access Control

The contract analysis system integrates seamlessly with the OpenAPI access control system:

```go
// Create OpenAPI generator with IP-based access control
config := &openapi.Config{
    Title:       "My API",
    Description: "API with comprehensive annotations and access control",
    Version:     "2.0.0",
    AccessControl: &openapi.AccessControlConfig{
        DefaultAccess:      openapi.AccessNonPublic, // Allow non-public IPs by default
        AllowedCIDRs:       []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
        EnableRateLimiter:  true,
        RateLimitRequests:  60,
        RateLimitWindow:    60,
        EnableAuditLogging: true,
    },
}

generator := openapi.NewGenerator(config)

// Add routes with annotations
generator.AddRouteWithAnnotation("GET", "/users/{id}", getUserHandler, userAnnotation)

// The generated handlers will automatically include access control
openapiHandler := generator.GetOpenAPIHandler()     // With IP filtering
swaggerHandler := generator.GetSwaggerUIHandler()   // With IP filtering
```

**Note:** The `AccessNonPublic` default now automatically includes Tailscale IPs (`100.64.0.0/10`), so you don't need to explicitly add them to the `AllowedCIDRs` list unless you want to be explicit.

## Access Control Presets

The system provides several access control presets:

```go
// Development - Allow all access
config := openapi.DevelopmentAccessControlConfig()

// Production - Non-public IPs only with rate limiting
config := openapi.ProductionAccessControlConfig()

// Strict - Only specific subnets allowed
config := openapi.StrictAccessControlConfig()

// Custom - Full control over access rules
config := &openapi.AccessControlConfig{
    DefaultAccess: openapi.AccessDenyAll,
    AllowedCIDRs: []string{"192.168.1.0/24", "10.0.0.0/24"},
    EnableRateLimiter: true,
    RateLimitRequests: 30,
    RateLimitWindow: 60,
}
```

## Future Enhancements

- [ ] Support for GraphQL schema generation
- [ ] Additional validation tag support
- [ ] Custom documentation generators
- [ ] Performance optimizations for large APIs
- [ ] Integration with API testing tools
- [ ] Advanced audit logging and analytics
- [ ] API key and OAuth integration for access control