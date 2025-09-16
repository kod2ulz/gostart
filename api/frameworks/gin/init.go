package gin

import (
	"github.com/kod2ulz/gostart/api"
)

// Setup registers the Gin router factory and sets it as the default
func Setup() {
	api.RegisterFramework("gin", NewRouter)
	api.SetDefaultFramework("gin")
}

// SetupWithOptions registers the Gin router factory with custom configuration
func SetupWithOptions(options func(*api.RouterConfig)) {
	// Wrap the Gin router factory to apply options
	api.RegisterFramework("gin", func(config *api.RouterConfig) (api.Router, error) {
		if options != nil {
			options(config)
		}
		return NewRouter(config)
	})
	api.SetDefaultFramework("gin")
}