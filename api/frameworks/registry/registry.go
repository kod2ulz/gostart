package registry

import (
	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/api/frameworks/gin"
)

// RegisterFrameworks registers all available frameworks
func RegisterFrameworks() {
	// Register Gin framework
	api.RegisterFramework("gin", gin.NewRouter)
	api.SetDefaultFramework("gin")
}

// SetupGin sets up the Gin framework with default configuration
func SetupGin() {
	gin.Setup()
}

// SetupGinWithOptions sets up the Gin framework with custom configuration
func SetupGinWithOptions(options func(*api.RouterConfig)) {
	gin.SetupWithOptions(options)
}