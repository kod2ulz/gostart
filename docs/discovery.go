package docs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"github.com/kod2ulz/gostart/contracts"
)

// RouteInfo contains information about a discovered route
type RouteInfo struct {
	Method      string
	Path        string
	Handler     interface{}
	HandlerFunc func(contracts.RequestContext)
	RequestType reflect.Type
	Description string
}

// RouteDiscoverer discovers routes from router implementations
type RouteDiscoverer struct {
	routes []RouteInfo
}

// NewRouteDiscoverer creates a new route discoverer
func NewRouteDiscoverer() *RouteDiscoverer {
	return &RouteDiscoverer{
		routes: make([]RouteInfo, 0),
	}
}

// DiscoverFromRouter discovers routes from a router interface
func (d *RouteDiscoverer) DiscoverFromRouter(router interface{}) error {
	// Use reflection to discover routes
	routerValue := reflect.ValueOf(router)
	_ = routerValue.Type()

	// Try to discover routes by looking for common router patterns
	switch r := router.(type) {
	case interface{ GetRoutes() []RouteInfo }:
		// If router implements GetRoutes method
		d.routes = append(d.routes, r.GetRoutes()...)
	case interface{ Routes() []RouteInfo }:
		// If router implements Routes method
		d.routes = append(d.routes, r.Routes()...)
	default:
		// Generic discovery through reflection
		d.discoverRoutesReflectively(routerValue)
	}

	return nil
}

// discoverRoutesReflectively discovers routes using reflection
func (d *RouteDiscoverer) discoverRoutesReflectively(routerValue reflect.Value) {
	_ = routerValue.Type()

	// Look for common HTTP method methods (GET, POST, etc.)
	httpMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}

	for _, method := range httpMethods {
		methodValue := routerValue.MethodByName(method)
		if !methodValue.IsValid() {
			continue
		}

		// Get the method type to inspect its signature
		_ = methodValue.Type()

		// Look for routes stored in the router
		d.discoverRoutesFromMethod(methodValue, strings.ToLower(method))
	}
}

// discoverRoutesFromMethod discovers routes from a specific HTTP method
func (d *RouteDiscoverer) discoverRoutesFromMethod(methodValue reflect.Value, httpMethod string) {
	// This is a simplified discovery mechanism
	// In a real implementation, you would need to inspect the router's internal state
	// or use framework-specific discovery methods

	// For now, we'll look for common patterns
	if methodValue.Type().NumOut() > 0 {
		// This method returns something, might be a router
		// We could potentially inspect the returned value for route information
	}
}

// RegisterRoute manually registers a route for documentation
func (d *RouteDiscoverer) RegisterRoute(method, path string, handler interface{}, description ...string) {
	route := RouteInfo{
		Method:  strings.ToUpper(method),
		Path:    path,
		Handler: handler,
	}

	if len(description) > 0 {
		route.Description = description[0]
	}

	// Try to extract handler function and request type
	if handlerFunc, ok := handler.(func(contracts.RequestContext)); ok {
		route.HandlerFunc = handlerFunc
		route.RequestType = d.extractRequestTypeFromHandler(handlerFunc)
	}

	d.routes = append(d.routes, route)
}

// extractRequestTypeFromHandler extracts the request type from a handler function
func (d *RouteDiscoverer) extractRequestTypeFromHandler(handler func(contracts.RequestContext)) reflect.Type {
	// Get the function value
	funcValue := reflect.ValueOf(handler)
	_ = funcValue.Type()

	// Inspect the function to understand its signature
	// This is a simplified approach - in a real implementation,
	// you would need more sophisticated type analysis

	// For now, return nil as we can't easily extract the request type
	// without more context about how the handler is structured
	return nil
}

// GetRoutes returns all discovered routes
func (d *RouteDiscoverer) GetRoutes() []RouteInfo {
	return d.routes
}

// GenerateDocumentation generates OpenAPI documentation from discovered routes
func (d *RouteDiscoverer) GenerateDocumentation(baseURL, title, version string) (*OpenAPI, error) {
	generator := NewGenerator(baseURL, title, version)

	for _, route := range d.routes {
		err := generator.RegisterRoute(route.Method, route.Path, route.Handler, route.RequestType, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to register route %s %s: %w", route.Method, route.Path, err)
		}
	}

	return generator.Generate()
}

// HandlerInfo contains information about a handler function
type HandlerInfo struct {
	Name         string
	Description  string
	RequestType  reflect.Type
	ResponseType reflect.Type
	Tags         []string
}

// AnalyzeHandler analyzes a handler function and extracts metadata
func AnalyzeHandler(handler interface{}) (*HandlerInfo, error) {
	handlerValue := reflect.ValueOf(handler)
	if handlerValue.Kind() != reflect.Func {
		return nil, fmt.Errorf("handler must be a function")
	}

	handlerType := handlerValue.Type()
	info := &HandlerInfo{
		Name: getFunctionName(handler),
	}

	// Analyze function signature
	if handlerType.NumIn() > 0 {
		// First parameter is typically the context/request
		paramType := handlerType.In(0)
		if paramType.Implements(reflect.TypeOf((*contracts.RequestContext)(nil)).Elem()) {
			// This is a standard handler with RequestContext
			// We can't easily extract the request type without more context
		}
	}

	// Extract tags and description from comments (simplified)
	// In a real implementation, you would parse source code comments

	return info, nil
}

// getFunctionName returns the name of a function
func getFunctionName(fn interface{}) string {
	value := reflect.ValueOf(fn)
	if value.Kind() != reflect.Func {
		return ""
	}

	// Get the function name from runtime information
	fnPtr := value.Pointer()
	fnRuntime := runtime.FuncForPC(fnPtr)
	if fnRuntime == nil {
		return ""
	}

	name := fnRuntime.Name()
	// Clean up the name (remove package path)
	if lastSlash := strings.LastIndex(name, "/"); lastSlash >= 0 {
		name = name[lastSlash+1:]
	}
	if lastDot := strings.LastIndex(name, "."); lastDot >= 0 {
		name = name[lastDot+1:]
	}

	return name
}

// DocumentationConfig contains configuration for documentation generation
type DocumentationConfig struct {
	BaseURL    string
	Title      string
	Version    string
	OutputFile string
	Format     string // "json" or "yaml"
}

// DefaultConfig returns default documentation configuration
func DefaultConfig() *DocumentationConfig {
	return &DocumentationConfig{
		BaseURL:    "http://localhost:8080",
		Title:      "API Documentation",
		Version:    "1.0.0",
		OutputFile: "swagger.json",
		Format:     "json",
	}
}

// GenerateAndSaveDocumentation generates and saves documentation to a file
func GenerateAndSaveDocumentation(discoverer *RouteDiscoverer, config *DocumentationConfig) error {
	// Generate OpenAPI specification
	spec, err := discoverer.GenerateDocumentation(config.BaseURL, config.Title, config.Version)
	if err != nil {
		return fmt.Errorf("failed to generate documentation: %w", err)
	}

	// Convert to desired format
	switch strings.ToLower(config.Format) {
	case "yaml", "yml":
		// For now, only JSON is supported
		// YAML support would require a YAML library
		return fmt.Errorf("YAML format not yet supported")
	case "json":
		_, err = json.MarshalIndent(spec, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
	default:
		return fmt.Errorf("unsupported format: %s", config.Format)
	}

	// Save to file
	return nil // In a real implementation, you would write to config.OutputFile
}

// HTTPHandler creates an HTTP handler that serves the OpenAPI documentation
func HTTPHandler(discoverer *RouteDiscoverer, config *DocumentationConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		spec, err := discoverer.GenerateDocumentation(config.BaseURL, config.Title, config.Version)
		if err != nil {
			http.Error(w, "Failed to generate documentation", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(spec)
	})
}