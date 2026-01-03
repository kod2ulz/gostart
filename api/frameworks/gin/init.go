package gin

import (
	"fmt"

	"github.com/kod2ulz/gostart/api"
)
var (
	ginIsSetup bool
)

// init automatically registers the Gin router factory when this package is imported
func init() {
	Setup()
}

// Setup registers the Gin router factory
// Note: This is now handled automatically by init(), but kept for backward compatibility
func Setup() {
	if ginIsSetup {
		fmt.Println("gin framework setup alaready run")
		return
	}
	fmt.Println("setting up gin framework")
	// Registration happens automatically via init()
	api.RegisterFramework("gin", func(config *api.RouterConfig) (api.Router, error) {
		return NewGinRouter(config)
	})
	api.SetDefaultFramework("gin")
	ginIsSetup = true
}

// SetupWithOptions registers the Gin router factory with custom configuration
// Note: This is now handled automatically by init(), but kept for backward compatibility
func SetupWithOptions(options ...any) {
	// Registration happens automatically via init()
}
