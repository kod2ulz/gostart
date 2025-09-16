package api

import (
	"fmt"
)

// registry manages router factory registration
var registry = struct {
	factories map[string]RouterFactory
	defaultFactory string
}{
	factories: make(map[string]RouterFactory),
}

// RegisterFramework registers a router factory for a specific framework
func RegisterFramework(name string, factory RouterFactory) {
	registry.factories[name] = factory
}

// SetDefaultFramework sets the default router factory
func SetDefaultFramework(name string) error {
	if _, exists := registry.factories[name]; !exists {
		return fmt.Errorf("router framework '%s' not registered", name)
	}
	registry.defaultFactory = name
	return nil
}

// DefaultFramework returns the name of the default framework
func DefaultFramework() string {
	return registry.defaultFactory
}

// GetRouterFactory gets the default router factory
func GetRouterFactory() (RouterFactory, error) {
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
func CreateRouter(config *RouterConfig) (Router, error) {
	factory, err := GetRouterFactory()
	if err != nil {
		return nil, err
	}
	return factory(config)
}

// ListFrameworks returns all registered framework names
func ListFrameworks() []string {
	names := make([]string, 0, len(registry.factories))
	for name := range registry.factories {
		names = append(names, name)
	}
	return names
}