package docs

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/kod2ulz/gostart/contracts"
)

// Example usage of the automated API documentation generation

// ExampleRequest demonstrates a request type with structured tags
type ExampleRequest struct {
	ID       string `json:"id" param:"id" description:"The unique identifier"`
	Name     string `json:"name" description:"The name field"`
	Email    string `json:"email" validate:"required,email" description:"User email address"`
	Age      int    `json:"age" query:"age" description:"User age"`
	Active   bool   `json:"active" query:"active" description:"Whether user is active"`
	APIKey   string `json:"-" header:"X-API-Key" description:"API key for authentication"`
	Internal string `json:"internal" description:"Internal field"`
}

// ExampleResponse demonstrates a response type
type ExampleResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Status string `json:"status"`
}

// ExampleHandler demonstrates a documented handler function
func ExampleHandler(ctx contracts.RequestContext) {
	// This handler would typically:
	// 1. Load request using DefaultRequestLoader
	// 2. Validate request
	// 3. Process business logic
	// 4. Return response

	// For documentation example, we'll just return success
	// Note: In a real handler, you would use app.RequestContext for JSON method
}

// NewExampleDocumentation creates a complete example documentation setup
func NewExampleDocumentation() (*HandlerRegistry, error) {
	// Create registry with custom configuration
	config := &DocumentationConfig{
		BaseURL:    "https://api.example.com/v1",
		Title:      "Example API Documentation",
		Version:    "1.0.0",
		OutputFile: "api-docs.json",
		Format:     "json",
	}

	registry := NewHandlerRegistry(config)

	// Register routes with documentation metadata
	registry.GET("/users/{id}", ExampleHandler).
		Description("Get user by ID").
		RequestType(reflect.TypeOf(ExampleRequest{}))

	registry.POST("/users", ExampleHandler).
		Description("Create a new user").
		RequestType(reflect.TypeOf(ExampleRequest{}))

	registry.PUT("/users/{id}", ExampleHandler).
		Description("Update an existing user").
		RequestType(reflect.TypeOf(ExampleRequest{}))

	registry.DELETE("/users/{id}", ExampleHandler).
		Description("Delete a user").
		RequestType(reflect.TypeOf(ExampleRequest{}))

	return registry, nil
}

// GenerateExampleDocumentation generates example documentation
func GenerateExampleDocumentation() ([]byte, error) {
	registry, err := NewExampleDocumentation()
	if err != nil {
		return nil, err
	}

	return registry.ToJSON()
}

// PrintExampleDocumentation prints the generated documentation
func PrintExampleDocumentation() {
	data, err := GenerateExampleDocumentation()
	if err != nil {
		fmt.Printf("Error generating documentation: %v\n", err)
		return
	}

	var prettyJSON map[string]interface{}
	if err := json.Unmarshal(data, &prettyJSON); err != nil {
		fmt.Printf("Raw documentation:\n%s\n", string(data))
		return
	}

	prettyData, err := json.MarshalIndent(prettyJSON, "", "  ")
	if err != nil {
		fmt.Printf("Raw documentation:\n%s\n", string(data))
		return
	}

	fmt.Printf("Generated OpenAPI Documentation:\n%s\n", string(prettyData))
}

// ExampleMiddlewareSetup shows how to integrate with existing router
func ExampleMiddlewareSetup() {
	// Create documentation middleware
	docConfig := &DocumentationConfig{
		BaseURL:    "http://localhost:8080",
		Title:      "My API",
		Version:    "1.0.0",
		OutputFile: "swagger.json",
	}

	docMiddleware := NewDocumentationMiddleware(docConfig)

	// Register handlers as you set up routes
	docMiddleware.RegisterHandler("GET", "/api/users", ExampleHandler, "Get all users")
	docMiddleware.RegisterHandler("POST", "/api/users", ExampleHandler, "Create a user")
	docMiddleware.RegisterHandler("GET", "/api/users/{id}", ExampleHandler, "Get user by ID")

	// Generate documentation
	_, err := docMiddleware.Generate()
	if err != nil {
		fmt.Printf("Error generating documentation: %v\n", err)
		return
	}

	// Save or serve the documentation
	fmt.Printf("Generated documentation with %d endpoints\n", len(docMiddleware.discoverer.routes))
}

// ComplexExample shows advanced usage with custom types
type ComplexRequest struct {
	Query     string                 `json:"query" query:"q" description:"Search query"`
	Filters   map[string]interface{} `json:"filters" description:"Search filters"`
	Pagination PaginationRequest     `json:"pagination"`
}

type PaginationRequest struct {
	Page  int `json:"page" query:"page" description:"Page number"`
	Limit int `json:"limit" query:"limit" description:"Items per page"`
}

type ComplexResponse struct {
	Results    []interface{} `json:"results"`
	Total      int            `json:"total"`
	Pagination PaginationResponse `json:"pagination"`
}

type PaginationResponse struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"totalPages"`
}

func ComplexHandler(ctx contracts.RequestContext) {
	// Complex handler implementation
	// Note: In a real handler, you would use app.RequestContext for JSON method
}

// NewComplexDocumentation shows documentation for complex types
func NewComplexDocumentation() (*OpenAPI, error) {
	registry := NewHandlerRegistry(DefaultConfig())

	registry.GET("/search", ComplexHandler).
		Description("Complex search with pagination").
		RequestType(reflect.TypeOf(ComplexRequest{}))

	return registry.GenerateDocumentation()
}

// IntegrationExample shows how to integrate with the app package router
func IntegrationExample() {
	/*
	// This would be integrated into your main application setup

	// Create documentation registry
	docRegistry := docs.NewHandlerRegistry(&docs.DocumentationConfig{
		BaseURL:    "http://localhost:8080",
		Title:      "My Application API",
		Version:    "1.0.0",
		OutputFile: "docs/swagger.json",
	})

	// Register routes as you define them
	app := app.Init(
		app.WithHandlerOverride("/api/users", func(ctx contracts.RequestContext) {
			// Your handler implementation
		}),
	)

	// Register for documentation
	docRegistry.GET("/api/users", yourHandler).
		Description("Get all users").
		RequestType(reflect.TypeOf(YourRequestType{}))

	// Generate documentation
	spec, err := docRegistry.GenerateDocumentation()
	if err != nil {
		log.Fatal("Failed to generate documentation:", err)
	}

	// Serve documentation endpoint
	app.Router().GET("/swagger.json", func(ctx contracts.RequestContext) {
		ctx.JSON(http.StatusOK, spec)
	})

	// Run the application
	app.Run()
	*/
}

// Demonstrates how the system works with the tag-driven request loading
func TagBasedLoadingExample() {
	/*
	The documentation system automatically reads struct tags to understand:

	- `json:"field"` - Request body field
	- `query:"param"` - URL query parameter
	- `param:"name"` - URL path parameter
	- `header:"X-Header"` - HTTP header
	- `description:"text"` - Field description
	- `validate:"required"` - Validation rules

	ExampleRequest struct {
		ID    string `json:"id" param:"id" description:"User ID"`
		Name  string `json:"name" validate:"required" description:"User name"`
		Email string `json:"email" validate:"required,email" description:"User email"`
		Age   int    `json:"age" query:"age" description:"User age (optional)"`
		Token string `json:"-" header:"Authorization" description:"Auth token"`
	}

	This would generate OpenAPI documentation that shows:
	- GET /users/{id} - path parameter "id"
	- POST /users - JSON body with name, email, age
	- Authorization header required
	- Field descriptions and validation rules
	*/
}