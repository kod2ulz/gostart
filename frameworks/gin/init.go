package gin

import (
	"github.com/kod2ulz/gostart/frameworks"
	"github.com/kod2ulz/gostart/router"
)

// Setup registers the Gin router factory and sets it as the default
func Setup() {
	frameworks.RegisterRouter("gin", NewRouter)
	frameworks.SetDefaultRouter("gin")
	frameworks.InitializeApp()
}

// SetupWithOptions registers the Gin router factory with custom configuration
func SetupWithOptions(options func(*router.RouterConfig)) {
	// Wrap the Gin router factory to apply options
	frameworks.RegisterRouter("gin", func(config interface{}) (router.Router, error) {
		if routerConfig, ok := config.(*router.RouterConfig); ok {
			if options != nil {
				options(routerConfig)
			}
			return NewRouter(routerConfig)
		}
		// Fallback for other config types - create default config
		defaultConfig := &router.RouterConfig{
			EnableRecovery: true,
			EnableLogging:  true,
		}
		return NewRouter(defaultConfig)
	})
	frameworks.SetDefaultRouter("gin")
	frameworks.InitializeApp()
}