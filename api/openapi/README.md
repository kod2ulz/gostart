# OpenAPI Documentation Generation

The OpenAPI package provides automatic API documentation generation for GoStart applications. It analyzes your routes, extracts type information from handlers, and generates comprehensive OpenAPI 3.0 specifications with minimal developer effort.

## Features

- **Automatic Route Discovery**: Routes are automatically registered with OpenAPI when added to the router
- **Intelligent Contract Analysis**: Extracts request/response contracts including path parameters, query strings, headers, and body schemas
- **Type Schema Generation**: Automatically generates JSON schemas from Go struct types with validation tags
- **Multiple UI Options**: Supports Swagger UI, Redoc, and other OpenAPI UI implementations
- **Framework Agnostic**: Works with any router implementation supporting the OpenAPIRouter interface
- **Error Response Documentation**: Automatically documents error responses based on handler return types
- **Customizable Annotations**: Add custom metadata to enhance API documentation

## Quick Start

### 1. Basic Usage

```go
package main

import (
    "log"

    "github.com/kod2ulz/gostart/api"
    gin_framework "github.com/kod2ulz/gostart/api/frameworks/gin"
    "github.com/kod2ulz/gostart/contracts"
)

func main() {
    // Create a router with automatic OpenAPI support
    config := api.DefaultRouterConfig()
    router, err := gin_framework.NewGinRouter(config)
    if err != nil {
        log.Fatal(err)
    }

    // Add routes - these are automatically documented
    router.GET("/users", func(ctx contracts.RequestContext) {
        if apiCtx, ok := ctx.(api.RequestContext); ok {
            apiCtx.JSON(200, map[string]interface{}{
                "users": []string{"user1", "user2"},
            })
        }
    })

    router.POST("/users", func(ctx contracts.RequestContext) {
        if apiCtx, ok := ctx.(api.RequestContext); ok {
            apiCtx.JSON(201, map[string]interface{}{
                "id":      "user3",
                "message": "User created",
            })
        }
    })

    // Start server with OpenAPI endpoints
    log.Println("Server starting on :8080")
    log.Println("OpenAPI JSON: http://localhost:8080/openapi.json")
    log.Println("Swagger UI: http://localhost:8080/swagger")

    router.Run(":8080")
}
```

### 2. Custom OpenAPI Configuration

```go
// Create custom OpenAPI configuration
openAPIConfig := &openapi.Info{
    Title:       "My API",
    Description: "This is a comprehensive API documentation",
    Version:     "2.0.0",
    Contact: &openapi.Contact{
        Name:  "API Support",
        Email: "support@example.com",
        URL:   "https://example.com/support",
    },
    License: &openapi.License{
        Name: "MIT",
        URL:  "https://opensource.org/licenses/MIT",
    },
}

// Create custom router configuration
config := &api.RouterConfig{
    EnableRecovery: true,
    EnableLogging:  true,
    StaticPaths: map[string]string{
        "/docs": "./docs",
    },
}

// Create router with custom configuration
router, err := gin_framework.NewGinRouter(config)
```

## OpenAPI UI Options

The system supports multiple OpenAPI UI implementations. You can choose the default Swagger UI or integrate with other UI tools.

### Default Swagger UI

The default Swagger UI is automatically served at `/swagger`:

```go
// Access Swagger UI at: http://localhost:8080/swagger
// OpenAPI JSON at: http://localhost:8080/openapi.json
```

### Custom UI Paths

You can customize the paths where documentation is served:

```go
// For Gin router, modify the addOpenAPIRoutes method
func (r *GinRouter) addOpenAPIRoutes() {
    // Custom OpenAPI JSON endpoint
    r.engine.GET("/api/docs/openapi.json", gin.WrapH(r.GetOpenAPIHandler()))

    // Custom Swagger UI endpoint
    r.engine.GET("/api/docs", gin.WrapH(r.GetSwaggerUIHandler()))

    // Alternative UI endpoint (e.g., Redoc)
    r.engine.GET("/api/docs/redoc", func(c *gin.Context) {
        c.HTML(200, "redoc.html", gin.H{
            "specURL": "/api/docs/openapi.json",
        })
    })
}
```

### Alternative UI Implementations

#### Redoc Integration

```go
// Create a Redoc handler
func redocHandler(specURL string) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>API Documentation</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
    <style>
        body { margin: 0; padding: 0; }
    </style>
</head>
<body>
    <redoc spec-url="%s"></redoc>
    <script src="https://cdn.jsdelivr.net/npm/redoc@latest/bundles/redoc.standalone.js"></script>
</body>
</html>`, specURL)

        w.Header().Set("Content-Type", "text/html")
        w.Write([]byte(html))
    })
}

// Add to router
router.GET("/redoc", gin.WrapH(redocHandler("/openapi.json")))
```

#### ReDoc Implementation

```go
// Create a ReDoc handler
func redocHandler(specURL string) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>ReDoc - API Documentation</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link href="https://unpkg.com/@stoplight/elements/styles.min.css" rel="stylesheet">
    <script src="https://unpkg.com/@stoplight/elements/web-components.min.js"></script>
</head>
<body>
    <elements-api
        apiDescriptionUrl="%s"
        router="hash"
        layout="sidebar"
    />
</body>
</html>`, specURL)

        w.Header().Set("Content-Type", "text/html")
        w.Write([]byte(html))
    })
}
```

#### Rapidoc Implementation

```go
// Create a RapiDoc handler
func rapidocHandler(specURL string) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>RapiDoc - API Documentation</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <script src="https://unpkg.com/rapidoc/dist/rapidoc-min.js"></script>
    <style>
        rapi-doc {
            height: 100vh;
            width: 100vw;
        }
    </style>
</head>
<body>
    <rapi-doc
        spec-url="%s"
        theme="light"
        render-style="read"
        show-header="false"
        allow-spec-url-load="false"
        allow-spec-file-load="false">
    </rapi-doc>
</body>
</html>`, specURL)

        w.Header().Set("Content-Type", "text/html")
        w.Write([]byte(html))
    })
}
```

## Advanced Features

### 1. Custom Annotations

You can add custom metadata to enhance your API documentation:

```go
// Create a custom annotation
annotation := openapi.Annotation{
    Summary:     "Get all users",
    Description: "Retrieves a list of all users in the system",
    Tags:        []string{"users", "administration"},
    OperationID: "getAllUsers",
    Deprecated:  false,
    Consumes:    []string{"application/json"},
    Produces:    []string{"application/json"},
    Parameters: []openapi.ParameterAnnotation{
        {
            Name:        "filter",
            In:          "query",
            Description: "Filter users by name",
            Required:    false,
            Schema: &openapi.Schema{
                Type: "string",
            },
        },
    },
}

// Register route with custom annotation
router.GET("/users", handler, annotation)
```

### 2. Struct-Based Schema Generation

The system automatically generates schemas from Go structs:

```go
type User struct {
    ID       string `json:"id" example:"123"`
    Name     string `json:"name" validate:"required" example:"John Doe"`
    Email    string `json:"email" validate:"required,email" example:"john@example.com"`
    Age      int    `json:"age" validate:"gte=18" example:"30"`
    Active   bool   `json:"active" default:"true"`
    CreatedAt string `json:"createdAt" readonly:"true"`
}

// The User struct will be automatically converted to:
// {
//   "type": "object",
//   "properties": {
//     "id": { "type": "string", "example": "123" },
//     "name": {
//       "type": "string",
//       "example": "John Doe",
//       "minLength": 1
//     },
//     "email": {
//       "type": "string",
//       "format": "email",
//       "example": "john@example.com"
//     },
//     "age": {
//       "type": "integer",
//       "minimum": 18,
//       "example": 30
//     },
//     "active": {
//       "type": "boolean",
//       "default": true
//     },
//     "createdAt": {
//       "type": "string",
//       "readOnly": true
//     }
//   },
//   "required": ["name", "email"]
// }
```

### 3. Error Response Documentation

The system automatically documents error responses based on your handler's return types:

```go
func getUserHandler(ctx contracts.RequestContext) (User, error) {
    // This handler will automatically generate:
    // - 200 response with User schema
    // - 400 response for validation errors
    // - 404 response for not found errors
    // - 500 response for server errors
}
```

### 4. Security Definitions

Add security requirements to your API:

```go
annotation := openapi.Annotation{
    Security: []map[string][]string{
        {"bearerAuth": {}},
    },
}
```

And define security schemes in your OpenAPI configuration:

```go
// In your router configuration
openAPIConfig.Components = &openapi.Components{
    SecuritySchemes: map[string]openapi.SecurityScheme{
        "bearerAuth": {
            Type:         "http",
            Scheme:       "bearer",
            BearerFormat: "JWT",
            Description:   "JWT Authorization header using the Bearer scheme.",
        },
    },
}
```

## Configuration Options

### Router Configuration

```go
config := &api.RouterConfig{
    // CORS settings
    AllowOrigins:     []string{"*"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
    AllowCredentials: true,
    MaxAge:           86400, // 24 hours

    // Middleware settings
    EnableRecovery:   true,
    EnableLogging:    true,

    // Static file serving
    StaticPaths: map[string]string{
        "/docs": "./docs",
        "/static": "./static",
    },
}
```

### OpenAPI Configuration

```go
openAPIConfig := &openapi.Info{
    Title:          "My API",
    Description:    "Comprehensive API documentation",
    Version:        "1.0.0",
    TermsOfService: "https://example.com/terms",

    Contact: &openapi.Contact{
        Name:  "API Team",
        Email: "api@example.com",
        URL:   "https://example.com/contact",
    },

    License: &openapi.License{
        Name: "Apache 2.0",
        URL:  "https://www.apache.org/licenses/LICENSE-2.0.html",
    },
}

// Add servers
servers := []openapi.Server{
    {
        URL:         "https://api.example.com/v1",
        Description: "Production server",
    },
    {
        URL:         "https://staging-api.example.com/v1",
        Description: "Staging server",
    },
}
```

## Testing Your OpenAPI Documentation

### 1. Access Endpoints

After starting your server, you can access:

- **OpenAPI JSON**: `http://localhost:8080/openapi.json`
- **Swagger UI**: `http://localhost:8080/swagger`
- **ReDoc**: `http://localhost:8080/redoc` (if configured)

### 2. Validate Your Specification

```go
// Generate and validate OpenAPI document
doc, err := router.GenerateOpenAPIDoc()
if err != nil {
    log.Fatal(err)
}

// Validate the document
if doc.OpenAPI != "3.0.0" {
    log.Fatal("Invalid OpenAPI version")
}

// Check required fields
if doc.Info.Title == "" || doc.Info.Version == "" {
    log.Fatal("Missing required OpenAPI info fields")
}
```

### 3. Export Documentation

```go
// Export to JSON
jsonData, err := json.MarshalIndent(doc, "", "  ")
if err != nil {
    log.Fatal(err)
}

// Save to file
err = os.WriteFile("openapi.json", jsonData, 0644)
if err != nil {
    log.Fatal(err)
}

// Export to YAML (using a YAML library)
yamlData, err := yaml.Marshal(doc)
if err != nil {
    log.Fatal(err)
}
err = os.WriteFile("openapi.yaml", yamlData, 0644)
```

## Best Practices

### 1. Use Descriptive Route Names

```go
// Good
router.GET("/users/:id", getUserHandler)
router.POST("/users", createUserHandler)
router.PUT("/users/:id", updateUserHandler)

// Avoid
router.GET("/u/:id", getUserHandler)
router.POST("/u", createUserHandler)
```

### 2. Add Comprehensive Comments

```go
// GetUser retrieves a user by ID
// @Summary Get user by ID
// @Description Retrieve detailed information about a specific user
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} User
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/{id} [get]
func getUserHandler(ctx contracts.RequestContext) {
    // Handler implementation
}
```

### 3. Use Struct Tags for Validation

```go
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=2,max=100"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"gte=18,lte=120"`
}
```

### 4. Organize API by Tags

```go
// Group related routes under tags
userAnnotation := openapi.Annotation{
    Tags: []string{"users"},
}

productAnnotation := openapi.Annotation{
    Tags: []string{"products"},
}

router.GET("/users", userHandler, userAnnotation)
router.POST("/products", productHandler, productAnnotation)
```

## Troubleshooting

### Common Issues

1. **Empty OpenAPI Documentation**
   - Ensure routes are added before accessing documentation
   - Check that the router implements the OpenAPIRouter interface

2. **Missing Schema Information**
   - Verify that your structs have proper JSON tags
   - Check that handler functions have proper return types

3. **UI Not Loading**
   - Ensure the documentation endpoints are properly configured
   - Check browser console for JavaScript errors

4. **CORS Issues**
   - Configure CORS settings in your router configuration
   - Ensure proper headers are set for API documentation

## Migration from Manual OpenAPI

If you're migrating from manually written OpenAPI specifications:

1. **Remove existing OpenAPI files** - The system generates these automatically
2. **Add struct tags** - Add JSON and validation tags to your data structures
3. **Update route handlers** - Ensure they return proper types for schema generation
4. **Configure UI** - Choose your preferred UI implementation and configure paths

## Next Steps

- [ ] Implement client SDK generation
- [ ] Add API versioning support
- [ ] Include response examples
- [ ] Add rate limiting documentation
- [ ] Implement webhook documentation

For more examples, see the `examples/` directory in the project repository.