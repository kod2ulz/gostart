package docs

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/kod2ulz/gostart/contracts"
)

// DocumentationMiddleware automatically registers routes for documentation generation
type DocumentationMiddleware struct {
	discoverer *RouteDiscoverer
	config     *DocumentationConfig
}

// NewDocumentationMiddleware creates a new documentation middleware
func NewDocumentationMiddleware(config *DocumentationConfig) *DocumentationMiddleware {
	if config == nil {
		config = DefaultConfig()
	}

	return &DocumentationMiddleware{
		discoverer: NewRouteDiscoverer(),
		config:     config,
	}
}

// RegisterHandler registers a handler function for documentation
func (m *DocumentationMiddleware) RegisterHandler(method, path string, handler interface{}, description ...string) {
	m.discoverer.RegisterRoute(method, path, handler, description...)
}

// RegisterRoute registers a route with metadata
func (m *DocumentationMiddleware) RegisterRoute(route RouteInfo) {
	m.discoverer.routes = append(m.discoverer.routes, route)
}

// Generate generates the OpenAPI documentation
func (m *DocumentationMiddleware) Generate() (*OpenAPI, error) {
	return m.discoverer.GenerateDocumentation(m.config.BaseURL, m.config.Title, m.config.Version)
}

// ToJSON converts the documentation to JSON
func (m *DocumentationMiddleware) ToJSON() ([]byte, error) {
	generator := NewGenerator(m.config.BaseURL, m.config.Title, m.config.Version)

	for _, route := range m.discoverer.GetRoutes() {
		err := generator.RegisterRoute(route.Method, route.Path, route.Handler, route.RequestType, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to register route %s %s: %w", route.Method, route.Path, err)
		}
	}

	return generator.ToJSON()
}

// RouteBuilder helps build route information with metadata
type RouteBuilder struct {
	route RouteInfo
}

// NewRouteBuilder creates a new route builder
func NewRouteBuilder(method, path string, handler interface{}) *RouteBuilder {
	return &RouteBuilder{
		route: RouteInfo{
			Method:  strings.ToUpper(method),
			Path:    path,
			Handler: handler,
		},
	}
}

// Description sets the route description
func (b *RouteBuilder) Description(desc string) *RouteBuilder {
	b.route.Description = desc
	return b
}

// RequestType sets the expected request type
func (b *RouteBuilder) RequestType(t reflect.Type) *RouteBuilder {
	b.route.RequestType = t
	return b
}

// HandlerFunc sets the handler function
func (b *RouteBuilder) HandlerFunc(fn func(contracts.RequestContext)) *RouteBuilder {
	b.route.HandlerFunc = fn
	return b
}

// Build returns the configured route
func (b *RouteBuilder) Build() RouteInfo {
	return b.route
}

// HandlerRegistry maintains a registry of documented handlers
type HandlerRegistry struct {
	middleware *DocumentationMiddleware
	routes     map[string]*RouteBuilder
}

// NewHandlerRegistry creates a new handler registry
func NewHandlerRegistry(config *DocumentationConfig) *HandlerRegistry {
	return &HandlerRegistry{
		middleware: NewDocumentationMiddleware(config),
		routes:     make(map[string]*RouteBuilder),
	}
}

// Register registers a handler with documentation metadata
func (r *HandlerRegistry) Register(method, path string, handler interface{}) *RouteBuilder {
	key := fmt.Sprintf("%s %s", strings.ToUpper(method), path)

	builder := NewRouteBuilder(method, path, handler)
	r.routes[key] = builder

	return builder
}

// GET registers a GET handler
func (r *HandlerRegistry) GET(path string, handler interface{}) *RouteBuilder {
	return r.Register("GET", path, handler)
}

// POST registers a POST handler
func (r *HandlerRegistry) POST(path string, handler interface{}) *RouteBuilder {
	return r.Register("POST", path, handler)
}

// PUT registers a PUT handler
func (r *HandlerRegistry) PUT(path string, handler interface{}) *RouteBuilder {
	return r.Register("PUT", path, handler)
}

// DELETE registers a DELETE handler
func (r *HandlerRegistry) DELETE(path string, handler interface{}) *RouteBuilder {
	return r.Register("DELETE", path, handler)
}

// PATCH registers a PATCH handler
func (r *HandlerRegistry) PATCH(path string, handler interface{}) *RouteBuilder {
	return r.Register("PATCH", path, handler)
}

// OPTIONS registers an OPTIONS handler
func (r *HandlerRegistry) OPTIONS(path string, handler interface{}) *RouteBuilder {
	return r.Register("OPTIONS", path, handler)
}

// HEAD registers a HEAD handler
func (r *HandlerRegistry) HEAD(path string, handler interface{}) *RouteBuilder {
	return r.Register("HEAD", path, handler)
}

// Build finalizes all registered routes
func (r *HandlerRegistry) Build() error {
	for _, builder := range r.routes {
		route := builder.Build()
		r.middleware.RegisterRoute(route)
	}
	return nil
}

// GenerateDocumentation generates the final documentation
func (r *HandlerRegistry) GenerateDocumentation() (*OpenAPI, error) {
	if err := r.Build(); err != nil {
		return nil, err
	}
	return r.middleware.Generate()
}

// ToJSON generates documentation as JSON
func (r *HandlerRegistry) ToJSON() ([]byte, error) {
	if err := r.Build(); err != nil {
		return nil, err
	}
	return r.middleware.ToJSON()
}

// AutoDiscover automatically discovers handlers from function names and comments
func AutoDiscover(pkgPath string) (*HandlerRegistry, error) {
	registry := NewHandlerRegistry(DefaultConfig())

	// This is a placeholder for auto-discovery functionality
	// In a real implementation, this would:
	// 1. Parse Go source files in the package
	// 2. Look for exported functions with specific signatures
	// 3. Extract documentation from comments
	// 4. Infer HTTP methods and paths from naming conventions

	return registry, nil
}

// CommentParser parses function comments for documentation metadata
type CommentParser struct {
	comments map[string]string
}

// NewCommentParser creates a new comment parser
func NewCommentParser() *CommentParser {
	return &CommentParser{
		comments: make(map[string]string),
	}
}

// ParseComments parses comments for a function
func (p *CommentParser) ParseComments(fn interface{}) (map[string]string, error) {
	// Get function information
	funcValue := reflect.ValueOf(fn)
	if funcValue.Kind() != reflect.Func {
		return nil, fmt.Errorf("expected function, got %T", fn)
	}

	// Get function name and try to find source comments
	_ = getFunctionName(fn)

	// This is a simplified approach - in a real implementation,
	// you would use go/parser to read actual source comments

	metadata := make(map[string]string)

	// Example comment format:
	// @Summary Get user by ID
	// @Description Retrieve a user's information
	// @Tags user
	// @Param id path string true "User ID"
	// @Success 200 {object} User
	// @Router /users/{id} [get]

	// For now, we'll return empty metadata
	return metadata, nil
}

// ExtractRouteInfoFromComments extracts route information from parsed comments
func ExtractRouteInfoFromComments(comments map[string]string) (method, path string, handlerInfo *HandlerInfo, err error) {
	// Parse comments to extract route information
	// This would parse @Router, @Summary, etc. tags

	// For now, return empty values
	return "", "", nil, nil
}
