package main

import (
	"log"

	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/api/contracts"
	"github.com/kod2ulz/gostart/api/openapi"
	"github.com/kod2ulz/gostart/contracts"
)

type User struct {
	ID    string `json:"id" example:"123" validate:"required"`
	Name  string `json:"name" validate:"required,min=2,max=100"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"gte=18,lte=120"`
}

type CreateUserRequest struct {
	Name  string `json:"name" validate:"required,min=2,max=100"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"gte=18,lte=120"`
}

func main() {
	// Create OpenAPI generator with access control
	config := &openapi.Config{
		Title:       "My API",
		Description: "API with comprehensive annotations and access control",
		Version:     "2.0.0",
		BaseURL:     "https://api.example.com",
		AccessControl: &openapi.AccessControlConfig{
			DefaultAccess:      openapi.AccessNonPublic, // Automatically includes private networks + Tailscale
			EnableRateLimiter:  true,
			RateLimitRequests:  60,
			RateLimitWindow:    60,
			EnableAuditLogging: true,
		},
	}

	generator := openapi.NewGenerator(config)

	// Example handlers
	getUserHandler := func(ctx contracts.RequestContext) (*User, error) {
		return &User{
			ID:    "123",
			Name:  "John Doe",
			Email: "john@example.com",
			Age:   30,
		}, nil
	}

	createUserHandler := func(ctx contracts.RequestContext) (*User, error) {
		return &User{
			ID:    "124",
			Name:  "Jane Doe",
			Email: "jane@example.com",
			Age:   25,
		}, nil
	}

	listUsersHandler := func(ctx contracts.RequestContext) ([]User, error) {
		return []User{
			{ID: "123", Name: "John Doe", Email: "john@example.com", Age: 30},
			{ID: "124", Name: "Jane Doe", Email: "jane@example.com", Age: 25},
		}, nil
	}

	// Create annotations
	getUserAnnotation := contracts.NewSummary("Get user by ID").
		WithDescription("Retrieve detailed information about a specific user").
		WithTags("users", "read").
		WithOperationID("getUser").
		WithParameter(contracts.NewParameter("id", "path", true)).
		WithResponse(200, contracts.NewResponse("User retrieved successfully")).
		WithResponse(404, contracts.NewResponse("User not found")).
		WithSecurity(map[string][]string{"bearerAuth": {}})

	createUserAnnotation := contracts.NewSummary("Create a new user").
		WithDescription("Create a new user account with the provided information").
		WithTags("users", "write").
		WithOperationID("createUser").
		WithSecurity(map[string][]string{"bearerAuth": {}}).
		WithConsumes("application/json").
		WithProduces("application/json").
		WithResponse(201, contracts.NewResponse("User created successfully")).
		WithResponse(400, contracts.NewResponse("Invalid request data"))

	listUsersAnnotation := contracts.NewSummary("List all users").
		WithDescription("Retrieve a paginated list of all users").
		WithTags("users", "read").
		WithOperationID("listUsers").
		WithParameter(contracts.NewParameter("page", "query", false)).
		WithParameter(contracts.NewParameter("pageSize", "query", false)).
		WithParameter(contracts.NewParameter("filter", "query", false)).
		WithSecurity(map[string][]string{"bearerAuth": {}}).
		WithResponse(200, contracts.NewResponse("List of users retrieved successfully"))

	// Add routes with annotations
	generator.AddRouteWithAnnotation("GET", "/users/{id}", getUserHandler, getUserAnnotation)
	generator.AddRouteWithAnnotation("POST", "/users", createUserHandler, createUserAnnotation)
	generator.AddRouteWithAnnotation("GET", "/users", listUsersHandler, listUsersAnnotation)

	// You can also add handler-level annotations
	handlerAnnotation := contracts.NewDescription("User management operations").
		WithTags("users")
	generator.AddHandlerAnnotation(getUserHandler, handlerAnnotation)
	generator.AddHandlerAnnotation(createUserHandler, handlerAnnotation)
	generator.AddHandlerAnnotation(listUsersHandler, handlerAnnotation)

	// Generate the OpenAPI specification
	doc, err := generator.Generate()
	if err != nil {
		log.Fatalf("Failed to generate OpenAPI specification: %v", err)
	}

	// Print summary
	log.Printf("OpenAPI Specification Generated:")
	log.Printf("  Title: %s", doc.Info.Title)
	log.Printf("  Version: %s", doc.Info.Version)
	log.Printf("  Paths: %d", len(doc.Paths))
	log.Printf("  Security: %t", generator.accessController != nil)

	// You can also access the handlers
	log.Printf("  OpenAPI Handler: Available with access control")
	log.Printf("  Swagger UI Handler: Available with access control")
}