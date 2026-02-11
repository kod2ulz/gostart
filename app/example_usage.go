package app

import (
	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/api/frameworks/gin"
)

// Example usage of the new architecture

// Basic initialization with Gin
func ExampleBasicInitialization() {
	// Initialize Gin as the router framework
	gin.Setup()

	// Now the app package will use Gin routers
	application := Init()
	application.Run()
}

// Custom initialization with options
func ExampleCustomInitialization() {
	// Initialize Gin with custom router configuration
	gin.SetupWithOptions(func(config *api.RouterConfig) {
		config.AllowOrigins = []string{"*"}
		config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE"}
		config.EnableRecovery = true
		config.EnableLogging = true
	})

	application := Init()
	application.Run()
}

// Framework-agnostic initialization example
func ExampleFrameworkAgnosticInitialization() {
	// This shows how you could switch to a different framework
	// For example, if you had an Echo implementation:

	// echo.Init() // Would set up Echo router factory

	// application := Init()
	// application.Run()
}

// With custom middleware
func ExampleWithMiddleware() {
	gin.SetupWithOptions(func(config *api.RouterConfig) {
		config.CustomMiddleware = []api.MiddlewareFunc{
			// Add custom middleware here
		}
	})

	application := Init()
	application.Run()
}

// Typical application setup
func ExampleTypicalSetup() {
	// 1. Initialize the router framework
	gin.Setup()

	// 2. Initialize the application
	application := Init(
		WithHeartbeatHandlers(),
		// Other initializers...
	)

	// 3. Run the application
	application.Run()
}
