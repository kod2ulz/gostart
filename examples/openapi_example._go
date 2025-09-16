package main

import (
	"log"

	"github.com/kod2ulz/gostart/api"
	gin_framework "github.com/kod2ulz/gostart/api/frameworks/gin"
	"github.com/kod2ulz/gostart/contracts"
)

func main() {
	// Create a new router with OpenAPI support
	config := api.DefaultRouterConfig()
	router, err := gin_framework.NewGinRouter(config)
	if err != nil {
		log.Fatal(err)
	}

	// Cast to OpenAPIRouter to access OpenAPI methods
	openAPIRouter, ok := router.(api.OpenAPIRouter)
	if !ok {
		log.Fatal("Router does not implement OpenAPIRouter interface")
	}

	// Add API routes - these will be automatically documented
	router.GET("/health", func(ctx contracts.RequestContext) {
		if apiCtx, ok := ctx.(api.RequestContext); ok {
			apiCtx.JSON(200, map[string]interface{}{
				"status": "healthy",
				"service": "go-start-api",
			})
		}
	})

	router.GET("/users", func(ctx contracts.RequestContext) {
		if apiCtx, ok := ctx.(api.RequestContext); ok {
			apiCtx.JSON(200, map[string]interface{}{
				"users": []map[string]interface{}{
					{"id": 1, "name": "John Doe", "email": "john@example.com"},
					{"id": 2, "name": "Jane Smith", "email": "jane@example.com"},
				},
			})
		}
	})

	router.POST("/users", func(ctx contracts.RequestContext) {
		if apiCtx, ok := ctx.(api.RequestContext); ok {
			apiCtx.JSON(201, map[string]interface{}{
				"id":      3,
				"message": "User created successfully",
			})
		}
	})

	router.GET("/users/:id", func(ctx contracts.RequestContext) {
		id := ctx.Param("id")
		if apiCtx, ok := ctx.(api.RequestContext); ok {
			apiCtx.JSON(200, map[string]interface{}{
				"id":    id,
				"name":  "John Doe",
				"email": "john@example.com",
			})
		}
	})

	// Print OpenAPI information
	doc, err := openAPIRouter.GenerateOpenAPIDoc()
	if err != nil {
		log.Printf("Warning: Could not generate OpenAPI doc: %v", err)
	} else {
		log.Printf("OpenAPI Documentation:")
		log.Printf("  Version: %s", doc.OpenAPI)
		log.Printf("  Title: %s", doc.Info.Title)
		log.Printf("  Version: %s", doc.Info.Version)
		log.Printf("  Paths: %d", len(doc.Paths))
	}

	// Start the server
	log.Println("Server starting on :8080")
	log.Println("OpenAPI JSON available at: http://localhost:8080/openapi.json")
	log.Println("Swagger UI available at: http://localhost:8080/swagger")

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}