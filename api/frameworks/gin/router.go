package gin

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/api/openapi"
	"github.com/kod2ulz/gostart/config"
	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/logr"
)

// GinRouter implements the api.Router interface using Gin
type GinRouter struct {
	engine          *gin.Engine
	group           *gin.RouterGroup
	openAPIRegistry *openapi.RouteRegistry
	openAPIConfig   *openapi.Info
	pathPrefix      string // Tracks the current path prefix from groups
	groupHierarchy  []string // Tracks the nested group structure for tag generation
	config          *api.RouterConfig
	schemas         map[string]*openapi.Schema // Registry for reusable schemas
	groupAnnotation *openapi.Annotation // Annotation for the current group level
}

// RequestContext implements both contracts.RequestContext and api.RequestContext
type RequestContext struct {
	*GinRequestContext
}

// hasNativeRecovery checks if native recovery middleware is already provided
func hasNativeRecovery(middlewares []any) bool {
	for _, mw := range middlewares {
		// Check if it's gin.Recovery or gin.CustomRecovery by function name
		mwValue := reflect.ValueOf(mw)
		if mwValue.Kind() == reflect.Func {
			fn := runtime.FuncForPC(mwValue.Pointer())
			if fn != nil {
				fnName := fn.Name()
				if strings.Contains(fnName, "Recovery") || strings.Contains(fnName, "CustomRecovery") {
					return true
				}
			}
		}
	}
	return false
}

// NewGinRouter creates a new Gin-based router
func NewGinRouter(config *api.RouterConfig) (api.Router, error) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	// Initialize OpenAPI registry
	openAPIRegistry := openapi.NewRouteRegistry()

	// Default OpenAPI configuration
	openAPIConfig := &openapi.Info{
		Title:       "GoStart API",
		Description: "API generated automatically by GoStart",
		Version:     "1.0.0",
	}

	// Apply native middleware first (e.g., gin.Recovery(), cors.New())
	if len(config.NativeMiddleware) > 0 {
		for _, mw := range config.NativeMiddleware {
			if handlerFunc, ok := mw.(gin.HandlerFunc); ok {
				engine.Use(handlerFunc)
			} else if handlerFunc, ok := mw.(func(*gin.Context)); ok {
				engine.Use(handlerFunc)
			} else {
				// Try to use it as-is (might be a middleware constructor that returns gin.HandlerFunc)
				logr.Log().Warn("Native middleware type not supported", "type", fmt.Sprintf("%T", mw))
			}
		}
	}

	// Apply default recovery if enabled and no native recovery middleware is provided
	hasNativeRecovery := hasNativeRecovery(config.NativeMiddleware)
	if config.EnableRecovery && !hasNativeRecovery {
		// If custom panic handler is provided, use custom recovery
		if config.PanicRecovery != nil {
			// Wrap custom handler to match gin's RecoveryFunc signature
			customRecovery := func(c *gin.Context, recovered any) {
				stack := debug.Stack()
				config.PanicRecovery(recovered, stack)
			}
			engine.Use(gin.CustomRecovery(customRecovery))
		} else {
			engine.Use(gin.Recovery())
		}
	}

	// Apply automatic logging middleware if enabled
	if config.EnableLogging {
		logConfig := config.LogConfig
		if logConfig == nil {
			logConfig = api.DefaultRequestLogConfig()
		}

		// Check config for logapi flag (defaulting to true)
		if shouldEnableLogging() {
			logger := logr.Log()
			// Skip logging if logger is not initialized (test environment)
			if logger != nil {
				loggingMiddleware := api.LoggingMiddleware(logger, logConfig)

				engine.Use(func(c *gin.Context) {
					ctx := &RequestContext{GinRequestContext: NewRequestContext(c).(*GinRequestContext)}
					if cont, err := loggingMiddleware(ctx); !cont || err != nil {
						if err != nil {
							c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]interface{}{
								"error": err.Error(),
							})
						}
						c.Abort()
						return
					}
					c.Next()
				})
			}
		}
	}

	// Apply custom middleware
	for _, mw := range config.CustomMiddleware {
		engine.Use(func(c *gin.Context) {
			ctx := &RequestContext{GinRequestContext: NewRequestContext(c).(*GinRequestContext)}
			if cont, err := mw(ctx); !cont || err != nil {
				if err != nil {
					c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]interface{}{
						"error": err.Error(),
					})
				}
				c.Abort()
				return
			}
			c.Next()
		})
	}

	router := &GinRouter{
		engine:          engine,
		openAPIRegistry: openAPIRegistry,
		openAPIConfig:   openAPIConfig,
		config:          config,
		schemas:         make(map[string]*openapi.Schema),
		groupHierarchy:  []string{},
	}

	// Configure static paths
	for path, root := range config.StaticPaths {
		router.engine.Static(path, root)
	}

	// Add OpenAPI documentation routes
	router.addOpenAPIRoutes()

	return router, nil
}

// HTTP Methods
func (r *GinRouter) GET(path string, handler api.HandlerFunc, options ...api.RouteOption) api.Router {
	return r.registerRoute("GET", path, handler, options...)
}

func (r *GinRouter) POST(path string, handler api.HandlerFunc, options ...api.RouteOption) api.Router {
	return r.registerRoute("POST", path, handler, options...)
}

func (r *GinRouter) PUT(path string, handler api.HandlerFunc, options ...api.RouteOption) api.Router {
	return r.registerRoute("PUT", path, handler, options...)
}

func (r *GinRouter) DELETE(path string, handler api.HandlerFunc, options ...api.RouteOption) api.Router {
	return r.registerRoute("DELETE", path, handler, options...)
}

func (r *GinRouter) PATCH(path string, handler api.HandlerFunc, options ...api.RouteOption) api.Router {
	return r.registerRoute("PATCH", path, handler, options...)
}

func (r *GinRouter) OPTIONS(path string, handler api.HandlerFunc, options ...api.RouteOption) api.Router {
	return r.registerRoute("OPTIONS", path, handler, options...)
}

func (r *GinRouter) HEAD(path string, handler api.HandlerFunc, options ...api.RouteOption) api.Router {
	return r.registerRoute("HEAD", path, handler, options...)
}

// Grouping
func (r *GinRouter) Group(path string, fn func(api.Router)) api.Router {
	// Create group from the current group, not from engine (to support nesting)
	var group *gin.RouterGroup
	if r.group != nil {
		group = r.group.Group(path)
	} else {
		group = r.engine.Group(path)
	}

	// Build the new path prefix
	newPrefix := r.pathPrefix
	if newPrefix == "" {
		newPrefix = "/" + path
	} else {
		newPrefix = newPrefix + "/" + path
	}
	// Ensure path starts with /
	if !strings.HasPrefix(newPrefix, "/") {
		newPrefix = "/" + newPrefix
	}

	// Build the new group hierarchy by appending this path segment
	newHierarchy := make([]string, len(r.groupHierarchy)+1)
	copy(newHierarchy, r.groupHierarchy)
	newHierarchy[len(newHierarchy)-1] = path

	subRouter := &GinRouter{
		engine:          r.engine,
		group:           group,
		openAPIRegistry: r.openAPIRegistry,
		openAPIConfig:   r.openAPIConfig,
		pathPrefix:      newPrefix,
		schemas:         r.schemas, // Share the schemas map
		config:          r.config,
		groupHierarchy:  newHierarchy,
	}
	fn(subRouter)
	return r
}

// AnnotateGroup sets an annotation for the current group level
// This annotation will be merged with all routes in this group
func (r *GinRouter) AnnotateGroup(annotation openapi.Annotation) api.Router {
	r.groupAnnotation = &annotation
	return r
}

// Middleware
func (r *GinRouter) Use(middleware ...api.MiddlewareFunc) api.Router {
	for _, mw := range middleware {
		r.currentGroup().Use(func(c *gin.Context) {
			ctx := &RequestContext{GinRequestContext: NewRequestContext(c).(*GinRequestContext)}
			if cont, err := mw(ctx); !cont || err != nil {
				if err != nil {
					c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]interface{}{
						"error": err.Error(),
					})
				}
				c.Abort()
				return
			}
			c.Next()
		})
	}
	return r
}

// Static files
func (r *GinRouter) StaticFile(path, filePath string) api.Router {
	r.engine.StaticFile(path, filePath)
	return r
}

func (r *GinRouter) Static(prefix, root string) api.Router {
	r.engine.Static(prefix, root)
	return r
}

// Raw access
func (r *GinRouter) Underlying() any {
	return r.engine
}

// Run server
func (r *GinRouter) Run(addr string) error {
	return r.engine.Run(addr)
}

// Helper methods
func (r *GinRouter) registerRoute(method, path string, handler api.HandlerFunc, options ...api.RouteOption) api.Router {
	fullPath := r.buildFullPath(path)

	// Process options
	var (
		middlewares []api.MiddlewareFunc
		annotation  openapi.Annotation
		handlers    []api.HandlerFunc
		requestType  reflect.Type
		responseType reflect.Type
		isList       bool
	)

	for _, opt := range options {
		switch v := opt.(type) {
		case api.RouteMiddleware:
			middlewares = append(middlewares, v.Middleware)
		case api.RouteAnnotation:
			annotation = v.Annotation
		case api.RouteHandler:
			handlers = append(handlers, v.Handler)
		case api.RouteType:
			requestType = v.RequestType
			responseType = v.ResponseType
			isList = v.IsList
		}
	}

	// Try to extract type information from the handler if it's a TypedHandlerFunc or TypedListHandlerFunc
	// This works even when TypedHandlerWithTypes is not used
	if requestType == nil {
		requestType, responseType, isList = r.extractTypesFromHandler(handler)
	}

	// Extract handler name for default summary and operationId
	handlerName := extractHandlerName(handler)
	if annotation.Summary == "" {
		annotation.Summary = handlerName
	}

	// Set operationId from handler name if not already provided
	if annotation.OperationID == "" {
		annotation.OperationID = toOperationID(handlerName)
	}

	// Auto-generate documentation from typed handler using reflection
	annotation = r.enhanceAnnotationFromHandler(handler, annotation)

	// If we have explicit type information, use it to enhance the annotation
	if requestType != nil {
		annotation = r.enhanceAnnotationFromTypes(annotation, requestType, responseType, isList, method, path)
	}

	// Deduplicate parameters by name (e.g., prevent ID from appearing twice)
	annotation.Parameters = r.deduplicateParameters(annotation.Parameters)

	// Merge with defaults (tags, etc.)
	annotation = r.mergeAnnotation(annotation, method, path)

	// Register with OpenAPI registry
	r.openAPIRegistry.Register(method, fullPath, handler, annotation)

	// Build the final handler chain
	finalHandler := handler
	for i := len(handlers) - 1; i >= 0; i-- {
		// Wrap handlers in reverse order so they execute in the right order
		// Use immediate function invocation to avoid closure capture issues
		currentHandler := handlers[i]
		nextHandler := finalHandler

		// Create a new closure with properly captured variables and panic recovery
		wrapped := func(ctx contracts.RequestContext) {
			defer func() {
				if recovered := recover(); recovered != nil {
					// Call custom panic handler if provided
					if r.config != nil && r.config.PanicRecovery != nil {
						stack := debug.Stack()
						r.config.PanicRecovery(recovered, stack)
					}

					// Return error response to client if we have a RequestContext
					if requestCtx, ok := ctx.(*RequestContext); ok {
						requestCtx.ctx.AbortWithStatusJSON(http.StatusInternalServerError, map[string]interface{}{
							"error":   "Internal server error",
							"message": fmt.Sprintf("Panic recovered: %v", recovered),
						})
					}
				}
			}()

			currentHandler(ctx)
			// Note: We can't easily check if context was aborted
			// Handlers should call ctx.Abort() if they want to stop the chain
			nextHandler(ctx)
		}
		finalHandler = wrapped
	}

	// Wrap with middleware
	wrappedHandler := r.wrapHandlerWithMiddleware(finalHandler, middlewares)

	// Register the route with Gin
	switch method {
	case "GET":
		r.currentGroup().GET(path, wrappedHandler)
	case "POST":
		r.currentGroup().POST(path, wrappedHandler)
	case "PUT":
		r.currentGroup().PUT(path, wrappedHandler)
	case "DELETE":
		r.currentGroup().DELETE(path, wrappedHandler)
	case "PATCH":
		r.currentGroup().PATCH(path, wrappedHandler)
	case "OPTIONS":
		r.currentGroup().OPTIONS(path, wrappedHandler)
	case "HEAD":
		r.currentGroup().HEAD(path, wrappedHandler)
	}

	return r
}

func (r *GinRouter) currentGroup() *gin.RouterGroup {
	if r.group != nil {
		return r.group
	}
	return &r.engine.RouterGroup
}

// handlePanic provides centralized panic recovery with optional custom handler
func (r *GinRouter) handlePanic(c *gin.Context, fn func()) {
	defer func() {
		if recovered := recover(); recovered != nil {
			// Call custom panic handler if provided
			if r.config != nil && r.config.PanicRecovery != nil {
				stack := debug.Stack()
				r.config.PanicRecovery(recovered, stack)
			}

			// Return error response to client
			c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]interface{}{
				"error":   "Internal server error",
				"message": fmt.Sprintf("Panic recovered: %v", recovered),
			})
		}
	}()

	fn()
}

// WrapHandler converts an api.HandlerFunc to gin.HandlerFunc for testing
// This is a convenience function for testing purposes
func WrapHandler(handler api.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := &RequestContext{GinRequestContext: NewRequestContext(c).(*GinRequestContext)}
		handler(ctx)
	}
}

// RequestContext implementations
func (ctx *RequestContext) Next() {
	ctx.ctx.Next()
}

func (ctx *RequestContext) Abort() {
	ctx.ctx.Abort()
}

func (ctx *RequestContext) AbortWithStatus(code int) {
	ctx.ctx.AbortWithStatus(code)
}

func (ctx *RequestContext) AbortWithStatusJSON(code int, obj any) {
	ctx.ctx.AbortWithStatusJSON(code, obj)
}

func (ctx *RequestContext) JSON(code int, obj any) {
	ctx.ctx.JSON(code, obj)
}

func (ctx *RequestContext) HTML(code int, name string, obj any) {
	ctx.ctx.HTML(code, name, obj)
}

func (ctx *RequestContext) String(code int, format string, values ...any) {
	ctx.ctx.String(code, format, values...)
}

func (ctx *RequestContext) Data(code int, contentType string, data []byte) {
	ctx.ctx.Data(code, contentType, data)
}

func (ctx *RequestContext) File(filepath string) {
	ctx.ctx.File(filepath)
}

func (ctx *RequestContext) SetHeader(key, value string) {
	ctx.ctx.Header(key, value)
}

func (ctx *RequestContext) Status(code int) {
	ctx.ctx.Status(code)
}

func (ctx *RequestContext) GetHeader(key string) string {
	return ctx.ctx.GetHeader(key)
}

func (ctx *RequestContext) SetCookie(cookie *http.Cookie) {
	http.SetCookie(ctx.ctx.Writer, cookie)
}

func (ctx *RequestContext) Cookie(name string) (string, error) {
	return ctx.ctx.Cookie(name)
}

func (ctx *RequestContext) ClientIP() string {
	return ctx.ctx.ClientIP()
}

// OpenAPIRouter interface implementation
func (r *GinRouter) GetOpenAPIHandler() http.Handler {
	return r.openAPIRegistry.GetOpenAPIJSONHandler("/openapi.json")
}

func (r *GinRouter) GetSwaggerUIHandler() http.Handler {
	return r.openAPIRegistry.GetSwaggerUIHandler("/swagger")
}

func (r *GinRouter) GenerateOpenAPIDoc() (*openapi.Document, error) {
	servers := []openapi.Server{
		{
			URL:         "http://localhost:8080",
			Description: "Development server",
		},
	}

	doc, err := r.openAPIRegistry.GenerateOpenAPIDoc(*r.openAPIConfig, servers)
	if err != nil {
		return nil, err
	}

	// Add all registered schemas to components/schemas
	if len(r.schemas) > 0 {
		if doc.Components.Schemas == nil {
			doc.Components.Schemas = make(map[string]openapi.Schema)
		}
		for name, schema := range r.schemas {
			doc.Components.Schemas[name] = *schema
		}
	}

	return doc, nil
}

func (r *GinRouter) SetOpenAPIInfo(info openapi.Info) {
	r.openAPIConfig = &info
}

// generateDefaultSummary generates a default summary for a route
func (r *GinRouter) generateDefaultSummary(method, path string) string {
	// Simple summary generation
	return fmt.Sprintf("%s %s", method, path)
}

// mergeAnnotation merges user-provided annotation with group annotation and auto-generated defaults
func (r *GinRouter) mergeAnnotation(annotation openapi.Annotation, method, path string) openapi.Annotation {
	fullPath := r.buildFullPath(path)
	defaultTags := r.generateTagsFromPath(fullPath)

	// Start with group annotation if present
	merged := openapi.Annotation{}
	if r.groupAnnotation != nil {
		merged = *r.groupAnnotation
	}

	// Merge route annotation (route annotation takes precedence)
	if annotation.Summary != "" {
		merged.Summary = annotation.Summary
	}
	if annotation.Description != "" {
		merged.Description = annotation.Description
	}
	if len(annotation.Tags) > 0 {
		merged.Tags = annotation.Tags
	}
	if annotation.Deprecated {
		merged.Deprecated = annotation.Deprecated
	}
	if len(annotation.Parameters) > 0 {
		merged.Parameters = annotation.Parameters
	}
	if annotation.RequestBody != nil {
		merged.RequestBody = annotation.RequestBody
	}
	if len(annotation.Responses) > 0 {
		merged.Responses = annotation.Responses
	}

	// If no summary provided, use default
	if merged.Summary == "" {
		merged.Summary = r.generateDefaultSummary(method, path)
	}

	// If no tags provided, use auto-generated tags
	if len(merged.Tags) == 0 {
		merged.Tags = defaultTags
	}

	return merged
}

// buildFullPath builds the full path by combining the prefix with the given path
func (r *GinRouter) buildFullPath(path string) string {
	if r.pathPrefix == "" {
		return "/" + path
	}
	// Remove leading slash from path if present, since prefix already has it
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	return r.pathPrefix + "/" + path
}

// generateTagsFromPath generates tags from the group hierarchy
// For grouped routes, this creates hierarchical tags that respect the group structure
// e.g., ["admin", "geo/attributes"] for router.Group("api", func(api) {
//   api.Group("admin", func(admin) {
//     admin.Group("geo", func(geo) {
//       geo.Group("attributes", ...)
func (r *GinRouter) generateTagsFromPath(path string) []string {
	// Use the group hierarchy if available
	if len(r.groupHierarchy) > 0 {
		// Skip "api" prefix if it's the first element
		startIdx := 0
		if len(r.groupHierarchy) > 0 && r.groupHierarchy[0] == "api" {
			startIdx = 1
		}

		// If we have groups after skipping "api", use them
		if len(r.groupHierarchy) > startIdx {
			// Use the first meaningful group as the primary tag (e.g., "admin")
			tags := []string{r.groupHierarchy[startIdx]}

			// For nested groups, combine subsequent groups with "/"
			// e.g., ["admin", "geo/attributes"] instead of ["admin", "geo", "attributes"]
			if len(r.groupHierarchy) > startIdx+2 {
				// Combine all groups after the first one
				nestedTag := strings.Join(r.groupHierarchy[startIdx+1:], "/")
				tags = append(tags, nestedTag)
			} else if len(r.groupHierarchy) > startIdx+1 {
				// Just one more group, add it as a separate tag
				tags = append(tags, r.groupHierarchy[startIdx+1])
			}

			return tags
		}
	}

	// Fallback to path-based tag generation for routes without groups
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")

	// Filter out empty strings and path parameters (starting with :)
	validParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" && !strings.HasPrefix(part, ":") {
			validParts = append(validParts, part)
		}
	}

	// If no tags, use "default"
	if len(validParts) == 0 {
		return []string{"default"}
	}

	// Skip "api" prefix if present
	startIdx := 0
	if len(validParts) > 0 && validParts[0] == "api" {
		startIdx = 1
	}

	// Build tags based on remaining path structure
	tags := make([]string, 0)
	if len(validParts) > startIdx {
		tags = append(tags, validParts[startIdx])

		if len(validParts) > startIdx+2 {
			nestedTag := strings.Join(validParts[startIdx+1:], "/")
			tags = append(tags, nestedTag)
		} else if len(validParts) > startIdx+1 {
			tags = append(tags, validParts[startIdx+1])
		}
	}

	// Fallback if we somehow have no tags
	if len(tags) == 0 {
		tags = []string{"default"}
	}

	return tags
}

// addOpenAPIRoutes adds routes for serving OpenAPI documentation
func (r *GinRouter) addOpenAPIRoutes() {
	// OpenAPI JSON endpoint
	r.engine.GET("/openapi.json", gin.WrapH(r.GetOpenAPIHandler()))

	// Swagger UI endpoint
	r.engine.GET("/swagger", gin.WrapH(r.GetSwaggerUIHandler()))

	// Redirect /swagger/ to /swagger
	r.engine.GET("/swagger/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger")
	})
}

// shouldEnableLogging checks if logging should be enabled based on config
func shouldEnableLogging() bool {
	// Default to true as requested by the user
	if value := config.Get("logapi"); value.Valid() {
		if strVal := value.String(); strVal == "false" || strVal == "0" {
			return false
		}
	}
	return true
}

// extractHandlerName extracts a clean handler name from the function
func extractHandlerName(handler api.HandlerFunc) string {
	// Get the function pointer and its name
	pc := reflect.ValueOf(handler).Pointer()
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "handler"
	}

	// Clean up the name
	name := fn.Name()

	// Remove package path
	parts := strings.Split(name, ".")
	if len(parts) > 0 {
		name = parts[len(parts)-1]
	}

	// Remove common suffixes like "-fm" (anonymous functions)
	name = regexp.MustCompile(`-fm\d*$`).ReplaceAllString(name, "")

	// Convert camelCase to Title Case for display
	name = regexp.MustCompile(`([a-z])([A-Z])`).ReplaceAllString(name, "$1 $2")
	name = strings.Title(name)

	return name
}

// toOperationID converts a handler name to an OpenAPI operationId format
// Converts "Search Categories" to "searchCategories"
func toOperationID(handlerName string) string {
	// Remove spaces and convert to camelCase
	words := strings.Fields(handlerName)
	if len(words) == 0 {
		return "operation"
	}

	// First word is lowercase
	result := strings.ToLower(words[0])

	// Subsequent words are capitalized (camelCase)
	for i := 1; i < len(words); i++ {
		word := words[i]
		if len(word) > 0 {
			// Capitalize first letter, lowercase the rest
			result += strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	return result
}

// enhanceAnnotationFromHandler uses reflection to auto-generate documentation from typed handlers
func (r *GinRouter) enhanceAnnotationFromHandler(handler api.HandlerFunc, annotation openapi.Annotation) openapi.Annotation {
	// Try to extract type information from the handler
	_ = reflect.ValueOf(handler)

	// Check if this is a typed handler (has generic type info embedded)
	// This is a placeholder for future enhancement where we inspect the handler's
	// parameter and response types using reflection
	//
	// For now, we'll rely on the OpenAPI registry which already does this inspection
	// when registering routes with typed handlers

	return annotation
}

// extractTypesFromHandler extracts request and response types from TypedHandlerFunc or TypedListHandlerFunc
func (r *GinRouter) extractTypesFromHandler(handler api.HandlerFunc) (requestType, responseType reflect.Type, isList bool) {
	handlerValue := reflect.ValueOf(handler)

	// Check if it's a TypedHandlerFunc or TypedListHandlerFunc by trying to call .Handler()
	if handlerValue.MethodByName("Handler").IsValid() {
		// This is likely a TypedHandlerFunc or TypedListHandlerFunc
		// Use reflection to extract the requestType, responseType, and isList fields
		typeField := handlerValue.Elem().FieldByName("requestType")
		respField := handlerValue.Elem().FieldByName("responseType")
		listField := handlerValue.Elem().FieldByName("isList")

		if typeField.IsValid() && respField.IsValid() && listField.IsValid() {
			// Extract the interface{} from reflect.Value and convert to reflect.Type
			if reqType, ok := typeField.Interface().(reflect.Type); ok {
				requestType = reqType
			}
			if resType, ok := respField.Interface().(reflect.Type); ok {
				responseType = resType
			}
			if isListVal, ok := listField.Interface().(bool); ok {
				isList = isListVal
			}
		}
	}

	// Try to extract type information from the handler function's signature
	// This works for TypedHandler closures that capture RequestModal[T]
	if requestType == nil && handlerValue.Kind() == reflect.Func {
		// Use runtime reflection to inspect the closure
		// Note: This is limited by Go's closure implementation
		// We can try to find captured RequestModal variables
		// but Go doesn't expose this easily

		// Alternative: Try to call the handler with a mock context to see what types it uses
		// This is complex and may have side effects

		// For now, we'll rely on TypedHandlerWithTypes for explicit type information
	}

	return
}

// deduplicateParameters removes duplicate parameters by name
// Keeps the first occurrence of each parameter
func (r *GinRouter) deduplicateParameters(params []openapi.ParameterAnnotation) []openapi.ParameterAnnotation {
	seen := make(map[string]bool)
	result := make([]openapi.ParameterAnnotation, 0, len(params))

	for _, param := range params {
		if !seen[param.Name] {
			seen[param.Name] = true
			result = append(result, param)
		}
	}

	return result
}

// registerSchema creates a reusable schema from a struct type and returns a $ref reference
func (r *GinRouter) registerSchema(t reflect.Type) *openapi.Schema {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil
	}

	// Generate schema name from type name
	typeName := t.Name()
	if typeName == "" {
		// Anonymous struct, generate a name based on path or hash
		typeName = fmt.Sprintf("Anonymous_%x", reflect.ValueOf(t).Pointer())
	}

	// Check if schema already exists in router's schema registry
	if _, exists := r.schemas[typeName]; exists {
		// Return existing schema reference
		return &openapi.Schema{
			Ref: "#/components/schemas/" + typeName,
		}
	}

	// Extract schema properties
	schema := r.extractRequestBodySchema(t)
	if schema == nil || len(schema.Properties) == 0 {
		return nil
	}

	// Store the schema in router
	r.schemas[typeName] = schema

	// Also add the schema directly to the OpenAPI generator's document
	// This ensures it's included when the JSON is served
	r.addSchemaToGenerator(typeName, schema)

	// Return a reference to the schema
	return &openapi.Schema{
		Ref: "#/components/schemas/" + typeName,
	}
}

// addSchemaToGenerator adds a schema to the OpenAPI generator's document
func (r *GinRouter) addSchemaToGenerator(name string, schema *openapi.Schema) {
	// Use the registry's AddSchema method to add the schema
	if r.openAPIRegistry != nil {
		r.openAPIRegistry.AddSchema(name, *schema)
	}
}

// enhanceAnnotationFromTypes enhances annotation with explicit type information
func (r *GinRouter) enhanceAnnotationFromTypes(annotation openapi.Annotation, requestType, responseType reflect.Type, isList bool, method, path string) openapi.Annotation {
	// Ensure response envelope schemas are registered
	r.registerResponseEnvelopeSchemas()

	// Extract fields from the request type to generate parameters
	if requestType != nil {
		// For GET requests, extract query and path parameters
		// Only extract if annotation doesn't already have parameters (e.g., from SearchableRouteDocs)
		if method == "GET" && len(annotation.Parameters) == 0 {
			// Extract query parameters from request struct fields
			queryParams := r.extractQueryParams(requestType)
			if len(queryParams) > 0 {
				// Append to existing parameters or create new list
				annotation.Parameters = append(annotation.Parameters, queryParams...)
			}
		}

		// Extract path parameters (fields with "param" tag)
		// Always extract these as they're not typically in annotations
		pathParams := r.extractPathParams(requestType)
		if len(pathParams) > 0 {
			annotation.Parameters = append(annotation.Parameters, pathParams...)
		}

		// For POST/PUT/PATCH, create request body schema from struct fields
		if method == "POST" || method == "PUT" || method == "PATCH" {
			// Register schema and get $ref
			schemaRef := r.registerSchema(requestType)

			if schemaRef != nil {
				// Create request body with the schema reference
				annotation.RequestBody = &openapi.RequestBody{
					Content: map[string]openapi.MediaType{
						"application/json": {
							Schema: schemaRef,
						},
					},
					Required: true,
				}
			}
		}
	}

	// Register response schema and enhance response documentation
	if responseType != nil {
		// For list responses, we need to create an array schema
		if isList {
			// responseType is the element type (R), not the slice ([]R)
			// Register the item type schema
			itemSchemaRef := r.registerSchema(responseType)

			if itemSchemaRef != nil {
				// Create array schema with $ref to items
				arraySchema := &openapi.Schema{
					Type:  "array",
					Items: itemSchemaRef,
				}

				// Register the array schema itself
				arraySchemaName := responseType.Name() + "List"
				if arraySchemaName == "List" {
					arraySchemaName = "AnonymousList"
				}
				r.schemas[arraySchemaName] = arraySchema
				r.addSchemaToGenerator(arraySchemaName, arraySchema)

				// Wrap in ResponseEnvelope with pagination metadata
				dataRef := "#/components/schemas/" + arraySchemaName
				wrappedSchema := r.wrapResponseEnvelope(dataRef, true)

				// Add or update the 200 response with the wrapped schema
				if annotation.Responses == nil {
					annotation.Responses = make(map[string]openapi.Response)
				}

				// Create success response with wrapped schema
				annotation.Responses["200"] = openapi.Response{
					Description: "Successful response",
					Content: map[string]openapi.MediaType{
						"application/json": {
							Schema: wrappedSchema,
						},
					},
				}
			}
		} else {
			// For single responses, register the response type schema
			responseSchemaRef := r.registerSchema(responseType)

			if responseSchemaRef != nil {
				// Wrap in ResponseEnvelope
				wrappedSchema := r.wrapResponseEnvelope(responseSchemaRef.Ref, false)

				// Add or update the 200 response with the schema
				if annotation.Responses == nil {
					annotation.Responses = make(map[string]openapi.Response)
				}

				// Create success response with wrapped schema
				annotation.Responses["200"] = openapi.Response{
					Description: "Successful response",
					Content: map[string]openapi.MediaType{
						"application/json": {
							Schema: wrappedSchema,
						},
					},
				}
			}
		}
	}

	// Add common error responses
	r.addErrorResponseSchemas(&annotation)

	return annotation
}

// registerResponseEnvelopeSchemas registers the response envelope wrapper schemas
func (r *GinRouter) registerResponseEnvelopeSchemas() {
	// Only register once
	if _, exists := r.schemas["ResponseEnvelope"]; exists {
		return
	}

	// Meta schema (pagination metadata)
	metaSchema := &openapi.Schema{
		Type: "object",
		Properties: map[string]openapi.Schema{
			"total": {
				Type:        "integer",
				Description: "Total number of items",
				Format:      "int64",
			},
			"limit": {
				Type:        "integer",
				Description: "Number of items per page",
			},
			"offset": {
				Type:        "integer",
				Description: "Number of items to skip",
			},
		},
	}
	r.schemas["Meta"] = metaSchema
	r.addSchemaToGenerator("Meta", metaSchema)

	// ErrorInfo schema
	errorInfoSchema := &openapi.Schema{
		Type: "object",
		Properties: map[string]openapi.Schema{
			"code": {
				Type:        "string",
				Description: "Error code",
			},
			"message": {
				Type:        "string",
				Description: "Error message",
			},
			"fields": {
				Type:        "object",
				Description: "Validation error fields",
				AdditionalProperties: &openapi.AdditionalProperties{
					Schema: &openapi.Schema{
						Type: "string",
					},
				},
			},
			"details": {
				Type:        "object",
				Description: "Additional error details",
			},
		},
		Required: []string{"code", "message"},
	}
	r.schemas["ErrorInfo"] = errorInfoSchema
	r.addSchemaToGenerator("ErrorInfo", errorInfoSchema)

	// ResponseEnvelope schema
	responseEnvelopeSchema := &openapi.Schema{
		Type: "object",
		Properties: map[string]openapi.Schema{
			"success": {
				Type:        "boolean",
				Description: "Whether the request was successful",
			},
			"type": {
				Type:        "string",
				Description: "Type of the response data",
			},
			"data": {
				Type:        "object",
				Description: "Response data (present on success)",
			},
			"references": {
				Type:        "object",
				Description: "Related entities",
			},
			"meta": {
				Description: "Pagination metadata",
				Ref:         "#/components/schemas/Meta",
			},
			"time": {
				Type:        "integer",
				Description: "Response timestamp",
				Format:      "int64",
			},
			"error": {
				Description: "Error information (present on failure)",
				Ref:         "#/components/schemas/ErrorInfo",
			},
		},
		Required: []string{"success", "type", "time"},
	}
	r.schemas["ResponseEnvelope"] = responseEnvelopeSchema
	r.addSchemaToGenerator("ResponseEnvelope", responseEnvelopeSchema)
}

// wrapResponseEnvelope wraps a data schema reference in a ResponseEnvelope
func (r *GinRouter) wrapResponseEnvelope(dataRef string, isList bool) *openapi.Schema {
	// Create an allOf schema that includes the ResponseEnvelope and the data schema
	// We'll use properties to create a proper wrapped schema

	wrappedSchema := &openapi.Schema{
		Type: "object",
		Properties: map[string]openapi.Schema{
			"success": {
				Type:        "boolean",
				Description: "Whether the request was successful",
				Example:     true,
			},
			"type": {
				Type:        "string",
				Description: "Type of the response data",
			},
			"data": {
				Ref: dataRef,
			},
			"time": {
				Type:        "integer",
				Description: "Response timestamp (Unix epoch)",
				Format:      "int64",
			},
		},
		Required: []string{"success", "type", "data", "time"},
	}

	// Add meta property for list responses (pagination)
	if isList {
		wrappedSchema.Properties["meta"] = openapi.Schema{
			Description: "Pagination metadata",
			Ref:         "#/components/schemas/Meta",
		}
		wrappedSchema.Required = append(wrappedSchema.Required, "meta")
	}

	return wrappedSchema
}

// addErrorResponseSchemas adds common error responses to the annotation
func (r *GinRouter) addErrorResponseSchemas(annotation *openapi.Annotation) {
	if annotation.Responses == nil {
		annotation.Responses = make(map[string]openapi.Response)
	}

	// Create error response schema (ResponseEnvelope with error field)
	errorResponseSchema := &openapi.Schema{
		Type: "object",
		Properties: map[string]openapi.Schema{
			"success": {
				Type:        "boolean",
				Description: "Whether the request was successful",
				Example:     false,
			},
			"type": {
				Type:        "string",
				Description: "Response type (error)",
				Example:     "error",
			},
			"time": {
				Type:        "integer",
				Description: "Response timestamp (Unix epoch)",
				Format:      "int64",
			},
			"error": {
				Description: "Error information",
				Ref:         "#/components/schemas/ErrorInfo",
			},
		},
		Required: []string{"success", "type", "error", "time"},
	}

	// 400 Bad Request (validation errors, invalid input)
	annotation.Responses["400"] = openapi.Response{
		Description: "Bad Request - Invalid input or validation error",
		Content: map[string]openapi.MediaType{
			"application/json": {
				Schema: errorResponseSchema,
			},
		},
	}

	// 401 Unauthorized (authentication required)
	annotation.Responses["401"] = openapi.Response{
		Description: "Unauthorized - Authentication required",
		Content: map[string]openapi.MediaType{
			"application/json": {
				Schema: errorResponseSchema,
			},
		},
	}

	// 403 Forbidden (insufficient permissions)
	annotation.Responses["403"] = openapi.Response{
		Description: "Forbidden - Insufficient permissions",
		Content: map[string]openapi.MediaType{
			"application/json": {
				Schema: errorResponseSchema,
			},
		},
	}

	// 404 Not Found
	annotation.Responses["404"] = openapi.Response{
		Description: "Not Found - Resource not found",
		Content: map[string]openapi.MediaType{
			"application/json": {
				Schema: errorResponseSchema,
			},
		},
	}

	// 500 Internal Server Error
	annotation.Responses["500"] = openapi.Response{
		Description: "Internal Server Error",
		Content: map[string]openapi.MediaType{
			"application/json": {
				Schema: errorResponseSchema,
			},
		},
	}
}

// extractRequestBodySchema extracts request body schema from a request struct
// Returns a schema with inline properties, skipping RequestModal[T] fields and path parameters
func (r *GinRouter) extractRequestBodySchema(t reflect.Type) *openapi.Schema {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return &openapi.Schema{Type: "object"}
	}

	schema := &openapi.Schema{
		Type:       "object",
		Properties: make(map[string]openapi.Schema),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}

		// Skip RequestModal[T] embedded fields specifically
		if field.Anonymous && strings.HasPrefix(field.Type.String(), "api.RequestModal[") {
			continue
		}

		// Skip path parameters (fields with "param" tag)
		if paramTag := field.Tag.Get("param"); paramTag != "" && paramTag != "-" {
			continue
		}

		// Get field name from JSON tag
		fieldName := field.Name
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" && jsonTag != "-" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				fieldName = parts[0]
			}
		} else if jsonTag == "-" {
			// Skip fields explicitly marked with "-"
			continue
		}

		// Determine field type and format
		fieldType, fieldFormat := r.openAPITypeFromReflect(field.Type)

		// Create field schema
		fieldSchema := openapi.Schema{
			Type: fieldType,
		}

		// Add format if present (e.g., "uuid" for UUID fields)
		if fieldFormat != "" {
			fieldSchema.Format = fieldFormat
		}

		// Extract description from description tag
		if descTag := field.Tag.Get("description"); descTag != "" {
			fieldSchema.Description = descTag
		} else {
			// Fallback to field name if no description
			fieldSchema.Description = fieldName
		}

		// Extract example from example tag
		if exampleTag := field.Tag.Get("example"); exampleTag != "" {
			fieldSchema.Example = r.parseExampleValue(exampleTag, field.Type)
		}

		// Check if field is required
		isRequired := false
		if validateTag := field.Tag.Get("validate"); validateTag != "" && strings.Contains(validateTag, "required") {
			isRequired = true
		}

		if isRequired {
			if schema.Required == nil {
				schema.Required = []string{}
			}
			schema.Required = append(schema.Required, fieldName)
		}

		schema.Properties[fieldName] = fieldSchema
	}

	return schema
}

// extractQueryParams extracts query parameters from a request struct
func (r *GinRouter) extractQueryParams(t reflect.Type) []openapi.ParameterAnnotation {
	var params []openapi.ParameterAnnotation

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return params
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}

		// Skip RequestModal[T] embedded fields
		if field.Anonymous && strings.HasPrefix(field.Type.String(), "api.RequestModal[") {
			continue
		}

		// Check for query tag
		queryTag := field.Tag.Get("query")
		if queryTag == "" || queryTag == "-" {
			continue
		}

		// Get parameter name from query tag
		paramName := strings.Split(queryTag, ",")[0]

		// Determine if required based on validate tag
		required := false
		if validateTag := field.Tag.Get("validate"); validateTag != "" {
			required = strings.Contains(validateTag, "required")
		}

		// Determine parameter type from field type
		paramType, paramFormat := r.openAPITypeFromReflect(field.Type)

		// Create schema for parameter
		schema := &openapi.Schema{
			Type: paramType,
		}

		// Add format if present
		if paramFormat != "" {
			schema.Format = paramFormat
		}

		// Extract description from description tag
		description := ""
		if descTag := field.Tag.Get("description"); descTag != "" {
			description = descTag
		}

		// Extract example from example tag
		if exampleTag := field.Tag.Get("example"); exampleTag != "" {
			schema.Example = r.parseExampleValue(exampleTag, field.Type)
		}

		params = append(params, openapi.ParameterAnnotation{
			Name:        paramName,
			In:          "query",
			Required:    required,
			Description: description,
			Schema:      schema,
		})
	}

	return params
}

// extractPathParams extracts path parameters from a request struct
func (r *GinRouter) extractPathParams(t reflect.Type) []openapi.ParameterAnnotation {
	var params []openapi.ParameterAnnotation

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return params
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}

		// Skip RequestModal[T] embedded fields
		if field.Anonymous && strings.HasPrefix(field.Type.String(), "api.RequestModal[") {
			continue
		}

		// Check for param tag
		paramTag := field.Tag.Get("param")
		if paramTag == "" || paramTag == "-" {
			continue
		}

		// Get parameter name from param tag
		paramName := strings.Split(paramTag, ",")[0]

		// Determine parameter type from field type
		paramType, paramFormat := r.openAPITypeFromReflect(field.Type)

		// Create schema for parameter
		schema := &openapi.Schema{
			Type: paramType,
		}

		// Add format if present
		if paramFormat != "" {
			schema.Format = paramFormat
		}

		// Extract description from description tag
		description := ""
		if descTag := field.Tag.Get("description"); descTag != "" {
			description = descTag
		}

		// Extract example from example tag
		if exampleTag := field.Tag.Get("example"); exampleTag != "" {
			schema.Example = r.parseExampleValue(exampleTag, field.Type)
		}

		params = append(params, openapi.ParameterAnnotation{
			Name:        paramName,
			In:          "path",
			Required:    true,
			Description: description,
			Schema:      schema,
		})
	}

	return params
}

// openAPITypeFromReflect converts a reflect.Type to OpenAPI type and format
// Returns (type, format) where format can be "uuid", "int64", "double", etc.
func (r *GinRouter) openAPITypeFromReflect(t reflect.Type) (string, string) {
	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Check for UUID type
	typeStr := t.String()
	if strings.HasSuffix(typeStr, "uuid.UUID") || strings.Contains(typeStr, "uuid.UUID]") {
		return "string", "uuid"
	}

	// Check for optional types from github.com/markphelps/optional
	if strings.HasPrefix(typeStr, "optional.") {
		// Extract the underlying type from optional.String, optional.Int, etc.
		typeName := t.Name()
		switch typeName {
		case "String":
			return "string", ""
		case "Int", "Int8", "Int16", "Int32", "Int64":
			return "integer", "int64"
		case "Uint", "Uint8", "Uint16", "Uint32", "Uint64":
			return "integer", "int64"
		case "Float32", "Float64":
			return "number", "double"
		case "Bool":
			return "boolean", ""
		default:
			// Try to extract underlying type from generic optional
			if t.Kind() == reflect.Struct {
				// optional types are structs, try to get the element type
				return "string", ""
			}
		}
	}

	// Check for pgtype types (nullable database types)
	if strings.HasPrefix(typeStr, "pgtype.") {
		typeName := t.Name()
		switch typeName {
		case "UUID":
			return "string", "uuid"
		case "Int4", "Int8":
			return "integer", "int64"
		case "Numeric":
			return "number", "double"
		case "Text", "Varchar":
			return "string", ""
		case "Bool":
			return "boolean", ""
		default:
			return "string", ""
		}
	}

	// Handle standard types
	switch t.Kind() {
	case reflect.String:
		return "string", ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer", "int64"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer", "int64"
	case reflect.Float32:
		return "number", "float"
	case reflect.Float64:
		return "number", "double"
	case reflect.Bool:
		return "boolean", ""
	default:
		return "string", ""
	}
}

// parseExampleValue parses an example tag string and converts it to the appropriate type
func (r *GinRouter) parseExampleValue(example string, fieldType reflect.Type) interface{} {
	// Handle pointer types
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

	// Parse based on the underlying type
	switch fieldType.Kind() {
	case reflect.String:
		return example
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Try to parse as integer
		var result int64
		if _, err := fmt.Sscanf(example, "%d", &result); err == nil {
			return result
		}
		return example
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// Try to parse as unsigned integer
		var result uint64
		if _, err := fmt.Sscanf(example, "%d", &result); err == nil {
			return result
		}
		return example
	case reflect.Float32, reflect.Float64:
		// Try to parse as float
		var result float64
		if _, err := fmt.Sscanf(example, "%f", &result); err == nil {
			return result
		}
		return example
	case reflect.Bool:
		// Parse as boolean
		if example == "true" {
			return true
		} else if example == "false" {
			return false
		}
		return example
	default:
		// For complex types or unknown types, return as string
		return example
	}
}

// wrapHandlerWithMiddleware wraps a handler with route-specific middleware
func (r *GinRouter) wrapHandlerWithMiddleware(handler api.HandlerFunc, middlewares []api.MiddlewareFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		r.handlePanic(c, func() {
			ctx := &RequestContext{GinRequestContext: NewRequestContext(c).(*GinRequestContext)}

			// Execute middleware in order
			for _, mw := range middlewares {
				cont, err := mw(ctx)
				if !cont || err != nil {
					if err != nil {
						c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]interface{}{
							"error": err.Error(),
						})
					}
					c.Abort()
					return
				}
			}

			// Execute the main handler
			handler(ctx)
		})
	}
}

// Factory function for creating Gin routers
var NewRouter = NewGinRouter
