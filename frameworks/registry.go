package frameworks

import (
	"fmt"

	"github.com/kod2ulz/gostart/router"
)

// RouterRegistry manages router factory registration
type RouterRegistry struct {
	factories map[string]router.RouterFactory
	defaultFactory string
}

var registry = &RouterRegistry{
	factories: make(map[string]router.RouterFactory),
}

// RegisterRouter registers a router factory for a specific framework
func RegisterRouter(name string, factory interface{}) {
	if routerFactory, ok := factory.(router.RouterFactory); ok {
		registry.factories[name] = routerFactory
	} else if customFactory, ok := factory.(func(interface{}) (router.Router, error)); ok {
		// Convert to RouterFactory
		registry.factories[name] = func(config *router.RouterConfig) (router.Router, error) {
			return customFactory(config)
		}
	}
}

// SetDefaultRouter sets the default router factory
func SetDefaultRouter(name string) error {
	if _, exists := registry.factories[name]; !exists {
		return fmt.Errorf("router factory '%s' not registered", name)
	}
	registry.defaultFactory = name
	return nil
}

// GetRouterFactory gets the default router factory
func GetRouterFactory() (router.RouterFactory, error) {
	if registry.defaultFactory == "" {
		return nil, fmt.Errorf("no default router factory set")
	}
	factory, exists := registry.factories[registry.defaultFactory]
	if !exists {
		return nil, fmt.Errorf("default router factory '%s' not found", registry.defaultFactory)
	}
	return factory, nil
}

// CreateRouter creates a router using the default factory
func CreateRouter(config *router.RouterConfig) (router.Router, error) {
	factory, err := GetRouterFactory()
	if err != nil {
		return nil, err
	}
	return factory(config)
}

// InitializeApp initializes the app package with the default router factory
func InitializeApp() {
	// No longer need to initialize app package - we use the frameworks package directly
}