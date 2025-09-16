package openapi

// OpenAPI documentation generation for GoStart
//
// This package provides automatic OpenAPI 3.0 specification generation from route definitions.
// It analyzes registered routes, extracts type information from handlers, and generates comprehensive
// API documentation with minimal developer effort.
//
// Features:
// - Automatic route discovery and documentation
// - Type schema generation from Go structs
// - Handler parameter and return type analysis
// - Structured annotations for additional metadata
// - Interactive Swagger UI integration
// - OpenAPI 3.0 specification compliance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/kod2ulz/gostart/api/contracts"
	globalConfig "github.com/kod2ulz/gostart/config"
)

// Document represents the complete OpenAPI 3.0 specification
type Document struct {
	OpenAPI    string                 `json:"openapi"`
	Info       Info                   `json:"info"`
	Servers    []Server               `json:"servers"`
	Paths      map[string]PathItem    `json:"paths"`
	Components Components             `json:"components"`
	Tags       []Tag                  `json:"tags,omitempty"`
}

// Info provides metadata about the API
type Info struct {
	Title          string            `json:"title"`
	Description    string            `json:"description,omitempty"`
	Version        string            `json:"version"`
	Contact        *Contact          `json:"contact,omitempty"`
	License        *License          `json:"license,omitempty"`
	TermsOfService string            `json:"termsOfService,omitempty"`
}

// Contact information for the exposed API
type Contact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// License information for the exposed API
type License struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

// Server represents a server
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// PathItem describes the operations available on a single path
type PathItem struct {
	Ref         string     `json:"$ref,omitempty"`
	Summary     string     `json:"summary,omitempty"`
	Description string     `json:"description,omitempty"`
	Get         *Operation `json:"get,omitempty"`
	Put         *Operation `json:"put,omitempty"`
	Post        *Operation `json:"post,omitempty"`
	Delete      *Operation `json:"delete,omitempty"`
	Options     *Operation `json:"options,omitempty"`
	Head        *Operation `json:"head,omitempty"`
	Patch       *Operation `json:"patch,omitempty"`
	Trace       *Operation `json:"trace,omitempty"`
	Servers     []Server   `json:"servers,omitempty"`
	Parameters  []Parameter `json:"parameters,omitempty"`
}

// Operation describes a single API operation on a path
type Operation struct {
	Tags         []string            `json:"tags,omitempty"`
	Summary      string              `json:"summary,omitempty"`
	Description  string              `json:"description,omitempty"`
	ExternalDocs *ExternalDocs       `json:"externalDocs,omitempty"`
	OperationID  string              `json:"operationId,omitempty"`
	Parameters   []Parameter         `json:"parameters,omitempty"`
	RequestBody  *RequestBody        `json:"requestBody,omitempty"`
	Responses    map[string]Response `json:"responses"`
	Callbacks     map[string]Callback `json:"callbacks,omitempty"`
	Deprecated   bool                `json:"deprecated,omitempty"`
	Security     []SecurityRequirement `json:"security,omitempty"`
	Servers      []Server            `json:"servers,omitempty"`
}

// ExternalDocs allows referencing an external resource for extended documentation
type ExternalDocs struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
}

// Parameter represents a parameter that can be used in operations
type Parameter struct {
	Name            string      `json:"name"`
	In              string      `json:"in"` // "query", "header", "path" or "cookie"
	Description     string      `json:"description,omitempty"`
	Required        bool        `json:"required"`
	Deprecated      bool        `json:"deprecated,omitempty"`
	AllowEmptyValue bool        `json:"allowEmptyValue,omitempty"`
	Style           string      `json:"style,omitempty"`
	Explode         bool        `json:"explode,omitempty"`
	AllowReserved   bool        `json:"allowReserved,omitempty"`
	Schema          *Schema     `json:"schema,omitempty"`
	Example         interface{} `json:"example,omitempty"`
	Examples        map[string]Example `json:"examples,omitempty"`
	Content         map[string]MediaType `json:"content,omitempty"`
}

// RequestBody describes a single request body
type RequestBody struct {
	Description string                 `json:"description,omitempty"`
	Content     map[string]MediaType   `json:"content"`
	Required    bool                   `json:"required,omitempty"`
}

// MediaType provides schema and examples for a media type
type MediaType struct {
	Schema   *Schema               `json:"schema,omitempty"`
	Example  interface{}           `json:"example,omitempty"`
	Examples map[string]Example    `json:"examples,omitempty"`
	Encoding map[string]Encoding  `json:"encoding,omitempty"`
}

// Encoding provides encoding information for a specific media type
type Encoding struct {
	ContentType   string            `json:"contentType,omitempty"`
	Headers       map[string]Header `json:"headers,omitempty"`
	Style         string            `json:"style,omitempty"`
	Explode       bool              `json:"explode,omitempty"`
	AllowReserved bool              `json:"allowReserved,omitempty"`
}

// Header represents a header parameter
type Header struct {
	Description     string        `json:"description,omitempty"`
	Required        bool          `json:"required"`
	Deprecated      bool          `json:"deprecated,omitempty"`
	AllowEmptyValue bool          `json:"allowEmptyValue,omitempty"`
	Style           string        `json:"style,omitempty"`
	Explode         bool          `json:"explode,omitempty"`
	AllowReserved   bool          `json:"allowReserved,omitempty"`
	Schema          *Schema       `json:"schema,omitempty"`
	Example         interface{}    `json:"example,omitempty"`
	Examples        map[string]Example `json:"examples,omitempty"`
	Content         map[string]MediaType `json:"content,omitempty"`
}

// Response describes a single response from an API Operation
type Response struct {
	Description string                 `json:"description"`
	Headers     map[string]Header      `json:"headers,omitempty"`
	Content     map[string]MediaType   `json:"content,omitempty"`
	Links       map[string]Link        `json:"links,omitempty"`
}

// Example represents an example
type Example struct {
	Summary       string      `json:"summary,omitempty"`
	Description   string      `json:"description,omitempty"`
	Value         interface{} `json:"value,omitempty"`
	ExternalValue string      `json:"externalValue,omitempty"`
}

// Link represents a link
type Link struct {
	OperationID string            `json:"operationId,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	Description string            `json:"description,omitempty"`
	Server       *Server           `json:"server,omitempty"`
}

// Callback represents a callback
type Callback struct {
	PathItem map[string]PathItem `json:"$ref,omitempty"`
}

// SecurityRequirement allows the definition of security requirements
type SecurityRequirement map[string][]string

// Tag represents a tag
type Tag struct {
	Name         string        `json:"name"`
	Description  string        `json:"description,omitempty"`
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty"`
}

// Schema represents a JSON Schema
type Schema struct {
	Ref                  string                 `json:"$ref,omitempty"`
	OneOf                []Schema               `json:"oneOf,omitempty"`
	AnyOf                []Schema               `json:"anyOf,omitempty"`
	AllOf                []Schema               `json:"allOf,omitempty"`
	Not                  *Schema                `json:"not,omitempty"`
	Type                 string                 `json:"type,omitempty"`
	Format               string                 `json:"format,omitempty"`
	Title                string                 `json:"title,omitempty"`
	Description          string                 `json:"description,omitempty"`
	Default              interface{}            `json:"default,omitempty"`
	Nullable             bool                   `json:"nullable,omitempty"`
	ReadOnly             bool                   `json:"readOnly,omitempty"`
	WriteOnly            bool                   `json:"writeOnly,omitempty"`
	Deprecated           bool                   `json:"deprecated,omitempty"`
	Example              interface{}            `json:"example,omitempty"`
	ExternalDocs         *ExternalDocs          `json:"externalDocs,omitempty"`
	Enum                 []interface{}          `json:"enum,omitempty"`
	Const                interface{}            `json:"const,omitempty"`
	MultipleOf           float64                `json:"multipleOf,omitempty"`
	Maximum              float64                `json:"maximum,omitempty"`
	ExclusiveMaximum     bool                   `json:"exclusiveMaximum,omitempty"`
	Minimum              float64                `json:"minimum,omitempty"`
	ExclusiveMinimum     bool                   `json:"exclusiveMinimum,omitempty"`
	MaxLength            int                    `json:"maxLength,omitempty"`
	MinLength            int                    `json:"minLength,omitempty"`
	Pattern              string                 `json:"pattern,omitempty"`
	MaxItems             int                    `json:"maxItems,omitempty"`
	MinItems             int                    `json:"minItems,omitempty"`
	UniqueItems          bool                   `json:"uniqueItems,omitempty"`
	MaxProperties        int                    `json:"maxProperties,omitempty"`
	MinProperties        int                    `json:"minProperties,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Items                *Schema                `json:"items,omitempty"`
	Properties           map[string]Schema      `json:"properties,omitempty"`
	AdditionalProperties *AdditionalProperties  `json:"additionalProperties,omitempty"`
}

// AdditionalProperties represents additional properties
type AdditionalProperties struct {
	SchemaRef *Schema `json:"$ref,omitempty"`
	Schema    *Schema `json:"schema,omitempty"`
	Bool      bool    `json:"bool,omitempty"`
}

// Components holds reusable objects
type Components struct {
	Schemas         map[string]Schema        `json:"schemas,omitempty"`
	Responses       map[string]Response      `json:"responses,omitempty"`
	Parameters      map[string]Parameter     `json:"parameters,omitempty"`
	Examples        map[string]Example       `json:"examples,omitempty"`
	RequestBodies   map[string]RequestBody   `json:"requestBodies,omitempty"`
	Headers         map[string]Header        `json:"headers,omitempty"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty"`
	Links           map[string]Link          `json:"links,omitempty"`
	Callbacks       map[string]Callback      `json:"callbacks,omitempty"`
}

// SecurityScheme allows the definition of security schemes
type SecurityScheme struct {
	Type             string            `json:"type"`
	Description      string            `json:"description,omitempty"`
	Name             string            `json:"name,omitempty"`
	In               string            `json:"in,omitempty"`
	Scheme           string            `json:"scheme,omitempty"`
	BearerFormat     string            `json:"bearerFormat,omitempty"`
	Flows            map[string]Flow   `json:"flows,omitempty"`
	OpenIdConnectUrl string            `json:"openIdConnectUrl,omitempty"`
}

// Flow represents a flow
type Flow struct {
	AuthorizationUrl string            `json:"authorizationUrl,omitempty"`
	TokenUrl         string            `json:"tokenUrl,omitempty"`
	RefreshUrl       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes,omitempty"`
}

// RouteInfo represents information about a registered route
type RouteInfo struct {
	Method      string
	Path        string
	Handler     interface{}
	HandlerType reflect.Type
	Middlewares []interface{}
	Annotations map[string]interface{}
}

// Annotation represents metadata that can be attached to routes
// This is defined later in the file with a more complete structure

// Generator handles OpenAPI specification generation
type Generator struct {
	doc               *Document
	routes            []RouteInfo
	typeCache         map[reflect.Type]*Schema
	annotations       map[string]Annotation
	config            *Config
	contractAnalyzer  *contracts.ContractAnalyzer
	annotationStore   *contracts.AnnotationStore
	accessController  *AccessController
}

// Config holds configuration for the OpenAPI generator
type Config struct {
	Title             string
	Description       string
	Version           string
	BaseURL           string
	Servers           []Server
	Tags              []Tag
	Contact           *Contact
	License           *License
	AccessControl     *AccessControlConfig
}

// NewGenerator creates a new OpenAPI generator
func NewGenerator(config *Config) *Generator {
	if config == nil {
		config = &Config{
			Title:       fmt.Sprintf("%s API", globalConfig.Get("app.name", "GoStart").String()),
			Description: "API documentation generated automatically",
			Version:     globalConfig.Get("version", "1.0.0").String(),
			BaseURL:     globalConfig.Get("app.base-url", fmt.Sprintf("http://%s:%s", globalConfig.Get("host", "localhost").String(), globalConfig.Get("port", "8080").String())).String(),
		}
	}

	// Initialize access controller if configured
	var accessController *AccessController
	if config.AccessControl != nil {
		accessController, _ = NewAccessController(config.AccessControl)
	}

	return &Generator{
		doc: &Document{
			OpenAPI: "3.0.0",
			Info: Info{
				Title:       config.Title,
				Description: config.Description,
				Version:     config.Version,
				Contact:     config.Contact,
				License:     config.License,
			},
			Servers: config.Servers,
			Paths:    make(map[string]PathItem),
			Components: Components{
				Schemas:   make(map[string]Schema),
				Responses: make(map[string]Response),
			},
			Tags: config.Tags,
		},
		typeCache:        make(map[reflect.Type]*Schema),
		annotations:      make(map[string]Annotation),
		config:           config,
		contractAnalyzer: contracts.NewContractAnalyzer(),
		annotationStore:  contracts.NewAnnotationStore(),
		accessController: accessController,
	}
}

// AddRoute registers a route for documentation generation
func (g *Generator) AddRoute(method, path string, handler interface{}, annotations map[string]interface{}) {
	handlerType := reflect.TypeOf(handler)

	// Convert annotations to our format
	annotation := Annotation{}
	if annotations != nil {
		if summary, ok := annotations["summary"].(string); ok {
			annotation.Summary = summary
		}
		if description, ok := annotations["description"].(string); ok {
			annotation.Description = description
		}
		if tags, ok := annotations["tags"].([]string); ok {
			annotation.Tags = tags
		}
		if deprecated, ok := annotations["deprecated"].(bool); ok {
			annotation.Deprecated = deprecated
		}
		// Store custom annotations
		annotation.Custom = annotations
	}

	route := RouteInfo{
		Method:      method,
		Path:        path,
		Handler:     handler,
		HandlerType: handlerType,
		Annotations: map[string]interface{}{
			"openapi": annotation,
		},
	}

	// Analyze request and response contracts
	if g.contractAnalyzer != nil {
		requestContract, err := g.contractAnalyzer.AnalyzeRequest(handler, path)
		if err == nil {
			route.Annotations["requestContract"] = requestContract
		}

		responseContract, err := g.contractAnalyzer.AnalyzeResponse(handler)
		if err == nil {
			route.Annotations["responseContract"] = responseContract
		}
	}

	g.routes = append(g.routes, route)

	// Add annotation to the annotation store if it's a new annotation
	if g.annotationStore != nil {
		g.annotationStore.AddRouteAnnotation(method, path, contracts.Annotation{
			Summary:     annotation.Summary,
			Description: annotation.Description,
			Tags:        annotation.Tags,
			Deprecated:  annotation.Deprecated,
		})
	}
}

// AddRouteWithAnnotation registers a route with a contract annotation
func (g *Generator) AddRouteWithAnnotation(method, path string, handler interface{}, annotation contracts.Annotation) {
	// Add to annotation store
	if g.annotationStore != nil {
		g.annotationStore.AddRouteAnnotation(method, path, annotation)
	}

	// Convert to internal annotation format
	internalAnnotation := Annotation{
		Summary:     annotation.Summary,
		Description: annotation.Description,
		OperationID: annotation.OperationID,
		Tags:        annotation.Tags,
		Deprecated:  annotation.Deprecated,
		Consumes:    annotation.Consumes,
		Produces:    annotation.Produces,
		Security:    annotation.Security,
		Custom:      annotation.Extensions,
	}

	g.AddRoute(method, path, handler, map[string]interface{}{
		"openapi": internalAnnotation,
	})
}

// AddHandlerAnnotation adds an annotation to a handler function
func (g *Generator) AddHandlerAnnotation(handler interface{}, annotation contracts.Annotation) {
	if g.annotationStore != nil {
		g.annotationStore.AddHandlerAnnotation(handler, annotation)
	}
}

// GetAnnotations retrieves combined annotations for a route and handler
func (g *Generator) GetAnnotations(method, path string, handler interface{}) []contracts.Annotation {
	if g.annotationStore == nil {
		return nil
	}
	return g.annotationStore.GetCombinedAnnotations(method, path, handler)
}

// GetOpenAPIHandler returns an HTTP handler that serves the OpenAPI specification
func (g *Generator) GetOpenAPIHandler() http.Handler {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		doc, err := g.Generate()
		if err != nil {
			http.Error(w, "Failed to generate OpenAPI spec: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(doc); err != nil {
			http.Error(w, "Failed to encode OpenAPI spec: "+err.Error(), http.StatusInternalServerError)
		}
	})

	// Apply access control if configured
	if g.accessController != nil {
		return g.accessController.Middleware()(handler)
	}

	return handler
}

// GetSwaggerUIHandler returns an HTTP handler that serves the Swagger UI
func (g *Generator) GetSwaggerUIHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve Swagger UI HTML
		html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Swagger UI</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.9.0/swagger-ui.css">
    <style>
        html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin: 0; background: #fafafa; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.9.0/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5.9.0/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: "` + r.URL.Path + `/../openapi.json",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout"
            });
        };
    </script>
</body>
</html>`

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})
}

// Generate produces the complete OpenAPI specification
func (g *Generator) Generate() (*Document, error) {
	// Process all routes
	for _, route := range g.routes {
		g.processRoute(route)
	}

	return g.doc, nil
}

// ToJSON converts the OpenAPI document to JSON
func (g *Generator) ToJSON() ([]byte, error) {
	doc, err := g.Generate()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(doc, "", "  ")
}

// processRoute analyzes a route and adds it to the OpenAPI document
func (g *Generator) processRoute(route RouteInfo) {
	pathItem, exists := g.doc.Paths[route.Path]
	if !exists {
		pathItem = PathItem{}
	}

	// Extract handler type information
	operation := g.createOperation(route)

	// Add operation to path item based on HTTP method
	switch strings.ToUpper(route.Method) {
	case "GET":
		pathItem.Get = operation
	case "POST":
		pathItem.Post = operation
	case "PUT":
		pathItem.Put = operation
	case "DELETE":
		pathItem.Delete = operation
	case "PATCH":
		pathItem.Patch = operation
	case "OPTIONS":
		pathItem.Options = operation
	case "HEAD":
		pathItem.Head = operation
	}

	g.doc.Paths[route.Path] = pathItem
}

// createOperation creates an OpenAPI operation from route information
func (g *Generator) createOperation(route RouteInfo) *Operation {
	annotation := route.Annotations["openapi"].(Annotation)

	operation := &Operation{
		Summary:     annotation.Summary,
		Description: annotation.Description,
		Tags:        annotation.Tags,
		Deprecated:  annotation.Deprecated,
		OperationID: strings.ToLower(route.Method) + route.Path,
		Responses:   g.createDefaultResponses(),
	}

	// Use contract information if available
	if requestContract, ok := route.Annotations["requestContract"].(*contracts.RequestContract); ok {
		g.enrichOperationFromRequestContract(operation, requestContract)
	}

	if responseContract, ok := route.Annotations["responseContract"].(*contracts.ResponseContract); ok {
		g.enrichOperationFromResponseContract(operation, responseContract)
	}

	// Analyze handler to extract parameter and response types
	g.analyzeHandler(route, operation)

	return operation
}

// enrichOperationFromRequestContract enriches operation with request contract information
func (g *Generator) enrichOperationFromRequestContract(operation *Operation, contract *contracts.RequestContract) {
	// Add path parameters
	for name, param := range contract.PathParameters {
		operation.Parameters = append(operation.Parameters, Parameter{
			Name:        name,
			In:          "path",
			Required:    param.Required,
			Description: param.Description,
			Schema: &Schema{
				Type:   param.Type,
				Format: param.Format,
			},
		})
	}

	// Add query parameters
	for name, param := range contract.QueryParameters {
		operation.Parameters = append(operation.Parameters, Parameter{
			Name:        name,
			In:          "query",
			Required:    param.Required,
			Description: param.Description,
			Schema: &Schema{
				Type:   param.Type,
				Format: param.Format,
			},
		})
	}

	// Add header parameters
	for name, param := range contract.Headers {
		operation.Parameters = append(operation.Parameters, Parameter{
			Name:        name,
			In:          "header",
			Required:    param.Required,
			Description: param.Description,
			Schema: &Schema{
				Type:   param.Type,
				Format: param.Format,
			},
		})
	}

	// Add cookie parameters
	for name, param := range contract.Cookies {
		operation.Parameters = append(operation.Parameters, Parameter{
			Name:        name,
			In:          "cookie",
			Required:    param.Required,
			Description: param.Description,
			Schema: &Schema{
				Type:   param.Type,
				Format: param.Format,
			},
		})
	}

	// Add request body if available
	if contract.Body != nil && contract.Body.Required {
		operation.RequestBody = &RequestBody{
			Content: map[string]MediaType{
				contract.Body.ContentType: {
					Schema: g.convertSchemaContract(contract.Body.Schema),
				},
			},
			Required: contract.Body.Required,
		}
	}
}

// enrichOperationFromResponseContract enriches operation with response contract information
func (g *Generator) enrichOperationFromResponseContract(operation *Operation, contract *contracts.ResponseContract) {
	statusCode := fmt.Sprintf("%d", contract.StatusCode)

	response := Response{
		Description: fmt.Sprintf("%d response", contract.StatusCode),
	}

	// Add response body if available
	if contract.Body != nil {
		response.Content = map[string]MediaType{
			contract.Body.ContentType: {
				Schema: g.convertSchemaContract(contract.Body.Schema),
			},
		}
	}

	// Add headers if available
	if len(contract.Headers) > 0 {
		response.Headers = make(map[string]Header)
		for name, header := range contract.Headers {
			response.Headers[name] = Header{
				Description: header.Description,
				Schema: &Schema{
					Type:   header.Type,
					Format: header.Format,
				},
			}
		}
	}

	operation.Responses[statusCode] = response

	// Add error responses if available
	for statusCode, errorResponse := range contract.ErrorResponses {
		errorStatusCode := fmt.Sprintf("%d", statusCode)
		operation.Responses[errorStatusCode] = Response{
			Description: fmt.Sprintf("%d error response", statusCode),
			Content: map[string]MediaType{
				errorResponse.ContentType: {
					Schema: g.convertSchemaContract(errorResponse.Body.Schema),
				},
			},
		}
	}
}

// convertSchemaContract converts a contracts.SchemaContract to openapi.Schema
func (g *Generator) convertSchemaContract(schemaContract *contracts.SchemaContract) *Schema {
	if schemaContract == nil {
		return nil
	}

	schema := &Schema{
		Type:                 schemaContract.Type,
		Format:               schemaContract.Format,
		Description:          schemaContract.Description,
		Nullable:             false,
		ReadOnly:             schemaContract.ReadOnly,
		WriteOnly:            schemaContract.WriteOnly,
		Deprecated:           schemaContract.Deprecated,
	}

	if schemaContract.Properties != nil {
		schema.Properties = make(map[string]Schema)
		for name, prop := range schemaContract.Properties {
			schema.Properties[name] = *g.convertSchemaContract(&prop)
		}
	}

	if schemaContract.Required != nil {
		schema.Required = schemaContract.Required
	}

	if schemaContract.Items != nil {
		schema.Items = g.convertSchemaContract(schemaContract.Items)
	}

	if schemaContract.AdditionalProperties {
		schema.AdditionalProperties = &AdditionalProperties{
			Bool: true,
		}
	}

	if schemaContract.Enum != nil {
		schema.Enum = schemaContract.Enum
	}

	if schemaContract.Default != nil {
		schema.Default = schemaContract.Default
	}

	if schemaContract.Example != nil {
		schema.Example = schemaContract.Example
	}

	if schemaContract.MinLength != nil {
		schema.MinLength = *schemaContract.MinLength
	}

	if schemaContract.MaxLength != nil {
		schema.MaxLength = *schemaContract.MaxLength
	}

	if schemaContract.Pattern != "" {
		schema.Pattern = schemaContract.Pattern
	}

	if schemaContract.Minimum != nil {
		schema.Minimum = *schemaContract.Minimum
	}

	if schemaContract.Maximum != nil {
		schema.Maximum = *schemaContract.Maximum
	}

	if schemaContract.ExclusiveMinimum {
		schema.ExclusiveMinimum = true
	}

	if schemaContract.ExclusiveMaximum {
		schema.ExclusiveMaximum = true
	}

	if schemaContract.MultipleOf != nil {
		schema.MultipleOf = *schemaContract.MultipleOf
	}

	if schemaContract.MinItems != nil {
		schema.MinItems = *schemaContract.MinItems
	}

	if schemaContract.MaxItems != nil {
		schema.MaxItems = *schemaContract.MaxItems
	}

	if schemaContract.UniqueItems {
		schema.UniqueItems = true
	}

	if schemaContract.MinProperties != nil {
		schema.MinProperties = *schemaContract.MinProperties
	}

	if schemaContract.MaxProperties != nil {
		schema.MaxProperties = *schemaContract.MaxProperties
	}

	return schema
}

// createDefaultResponses creates default response schemas
func (g *Generator) createDefaultResponses() map[string]Response {
	return map[string]Response{
		"200": {
			Description: "Successful response",
			Content: map[string]MediaType{
				"application/json": {
					Schema: &Schema{
						Type: "object",
						Properties: map[string]Schema{
							"success": {Type: "boolean"},
							"data":    {Type: "object"},
							"time":    {Type: "integer", Format: "int64"},
						},
					},
				},
			},
		},
		"400": {
			Description: "Bad request",
			Content: map[string]MediaType{
				"application/json": {
					Schema: &Schema{
						Type: "object",
						Properties: map[string]Schema{
							"success": {Type: "boolean"},
							"error":   {Type: "object"},
						},
					},
				},
			},
		},
	}
}

// analyzeHandler extracts type information from the handler function
func (g *Generator) analyzeHandler(route RouteInfo, operation *Operation) {
	handlerType := route.HandlerType

	if handlerType.Kind() == reflect.Func {
		// Analyze function parameters and return values
		g.analyzeFunctionType(handlerType, operation, route)
	}
}

// analyzeFunctionType analyzes a function type to extract parameter and response types
func (g *Generator) analyzeFunctionType(funcType reflect.Type, operation *Operation, route RouteInfo) {
	// This is a simplified implementation
	// In a real implementation, we would:
	// 1. Analyze input parameters to extract request body schemas
	// 2. Analyze return values to extract response schemas
	// 3. Extract path parameters from the route path
	// 4. Generate schema definitions for custom types

	// For now, let's extract path parameters
	pathParams := g.extractPathParameters(route.Path)
	if len(pathParams) > 0 {
		operation.Parameters = append(operation.Parameters, pathParams...)
	}
}

// extractPathParameters extracts path parameters from a route path
func (g *Generator) extractPathParameters(path string) []Parameter {
	var params []Parameter

	// Simple path parameter extraction
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			paramName := strings.TrimPrefix(part, ":")
			params = append(params, Parameter{
				Name:     paramName,
				In:       "path",
				Required: true,
				Schema: &Schema{
					Type: "string",
				},
			})
		}
	}

	return params
}

// RegisterRoute is a convenience function for adding routes with annotations
func RegisterRoute(method, path string, handler interface{}, annotations map[string]interface{}) {
	// This would typically be called by the router when routes are registered
	// For now, it's a placeholder that would be integrated with the router
	_ = method
	_ = path
	_ = handler
	_ = annotations
}

// Annotation represents metadata for a route or operation
type Annotation struct {
	Summary     string                 `json:"summary,omitempty"`
	Description string                 `json:"description,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	OperationID string                 `json:"operationId,omitempty"`
	Deprecated  bool                   `json:"deprecated,omitempty"`
	Consumes    []string               `json:"consumes,omitempty"`
	Produces    []string               `json:"produces,omitempty"`
	Parameters  []ParameterAnnotation `json:"parameters,omitempty"`
	Responses   map[string]Response     `json:"responses,omitempty"`
	Security    []map[string][]string  `json:"security,omitempty"`
	External    *ExternalDocs          `json:"externalDocs,omitempty"`
	Custom      map[string]interface{} `json:"custom,omitempty"`
}

// ParameterAnnotation represents parameter-specific metadata
type ParameterAnnotation struct {
	Name        string      `json:"name"`
	In          string      `json:"in"`
	Description string      `json:"description,omitempty"`
	Required    bool        `json:"required"`
	Schema      *Schema     `json:"schema,omitempty"`
	Example     interface{} `json:"example,omitempty"`
	Deprecated  bool        `json:"deprecated,omitempty"`
}

// ResponseAnnotation represents response-specific metadata
type ResponseAnnotation struct {
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
	Headers     map[string]Header `json:"headers,omitempty"`
	Examples    map[string]Example `json:"examples,omitempty"`
}

// TagAnnotation represents tag metadata
type TagAnnotation struct {
	Name         string        `json:"name"`
	Description  string        `json:"description,omitempty"`
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty"`
}

// SecurityAnnotation represents security scheme metadata
type SecurityAnnotation struct {
	Type             string            `json:"type"`
	Description      string            `json:"description,omitempty"`
	Name             string            `json:"name,omitempty"`
	In               string            `json:"in,omitempty"`
	Scheme           string            `json:"scheme,omitempty"`
	BearerFormat     string            `json:"bearerFormat,omitempty"`
	Flows            map[string]OAuthFlow `json:"flows,omitempty"`
	OpenIdConnectUrl string            `json:"openIdConnectUrl,omitempty"`
}

// OAuthFlow represents OAuth flow configuration
type OAuthFlow struct {
	AuthorizationUrl string            `json:"authorizationUrl,omitempty"`
	TokenUrl        string            `json:"tokenUrl,omitempty"`
	RefreshUrl      string            `json:"refreshUrl,omitempty"`
	Scopes          map[string]string `json:"scopes"`
}

// RouteRegistry manages route registration and annotation storage
type RouteRegistry struct {
	routes      map[string]RouteInfo
	annotations map[string]Annotation
	generator   *Generator
	mu          sync.RWMutex
}

// NewRouteRegistry creates a new route registry
func NewRouteRegistry() *RouteRegistry {
	return &RouteRegistry{
		routes:      make(map[string]RouteInfo),
		annotations: make(map[string]Annotation),
		generator:   NewGenerator(&Config{}),
	}
}

// Register registers a route with its annotations
func (r *RouteRegistry) Register(method, path string, handler interface{}, annotation Annotation) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%s:%s", method, path)

	// Extract route information from handler
	routeInfo := r.extractRouteInfo(method, path, handler)
	r.routes[key] = routeInfo

	// Store annotations
	if annotation.Summary == "" {
		annotation.Summary = r.generateSummary(method, path)
	}
	r.annotations[key] = annotation

	// Register with generator
	annotationsMap := make(map[string]interface{})
	// Convert annotation to map
	annotationValue := reflect.ValueOf(annotation)
	annotationType := annotationValue.Type()
	for i := 0; i < annotationValue.NumField(); i++ {
		field := annotationType.Field(i)
		fieldValue := annotationValue.Field(i)
		if !fieldValue.IsZero() {
			annotationsMap[field.Name] = fieldValue.Interface()
		}
	}
	r.generator.AddRoute(method, path, handler, annotationsMap)
}

// extractRouteInfo extracts route information from handler function
func (r *RouteRegistry) extractRouteInfo(method, path string, handler interface{}) RouteInfo {
	handlerValue := reflect.ValueOf(handler)
	handlerType := handlerValue.Type()

	return RouteInfo{
		Method:      method,
		Path:        path,
		Handler:     handler,
		HandlerType: handlerType,
	}
}

// generateSummary generates a default summary for a route
func (r *RouteRegistry) generateSummary(method, path string) string {
	// Convert path to a more readable format
	summary := strings.ToUpper(method) + " " + path

	// Remove parameter placeholders for cleaner summary
	summary = strings.ReplaceAll(summary, "/:", "/")

	// Split by / and take the last meaningful part
	parts := strings.Split(summary, "/")
	if len(parts) > 1 {
		// Rebuild a cleaner summary
		var cleanParts []string
		for _, part := range parts {
			if part != "" && !strings.HasPrefix(part, ":") {
				cleanParts = append(cleanParts, part)
			}
		}
		if len(cleanParts) > 0 {
			summary = strings.Join(cleanParts[len(cleanParts)-1:], " ")
		}
	}

	return summary
}

// GetRoute returns a registered route by method and path
func (r *RouteRegistry) GetRoute(method, path string) (RouteInfo, Annotation, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", method, path)
	route, routeExists := r.routes[key]
	annotation, annotationExists := r.annotations[key]

	return route, annotation, routeExists && annotationExists
}

// GetAllRoutes returns all registered routes
func (r *RouteRegistry) GetAllRoutes() map[string]struct {
	Route    RouteInfo
	Annotation Annotation
} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]struct {
		Route    RouteInfo
		Annotation Annotation
	})

	for key, route := range r.routes {
		annotation := r.annotations[key]
		result[key] = struct {
			Route    RouteInfo
			Annotation Annotation
		}{
			Route:      route,
			Annotation: annotation,
		}
	}

	return result
}

// GenerateOpenAPIDoc generates OpenAPI documentation from all registered routes
func (r *RouteRegistry) GenerateOpenAPIDoc(info Info, servers []Server) (*Document, error) {
	// Set the info and servers in the generator's document
	r.generator.doc.Info = info
	r.generator.doc.Servers = servers
	return r.generator.Generate()
}

// GetSwaggerUIHandler returns a handler for serving Swagger UI
func (r *RouteRegistry) GetSwaggerUIHandler(basePath string) http.Handler {
	return r.generator.SwaggerUI(basePath)
}

// GetOpenAPIJSONHandler returns a handler for serving OpenAPI JSON
func (r *RouteRegistry) GetOpenAPIJSONHandler(basePath string) http.Handler {
	return r.generator.OpenAPIJSON(basePath)
}

// DefaultRouteRegistry is the global instance for convenience
var DefaultRouteRegistry = NewRouteRegistry()

// Register is a convenience function for registering with the default registry
func Register(method, path string, handler interface{}, annotation Annotation) {
	DefaultRouteRegistry.Register(method, path, handler, annotation)
}

// Get is a convenience function for GET routes
func Get(path string, handler interface{}, annotation Annotation) {
	Register("GET", path, handler, annotation)
}

// Post is a convenience function for POST routes
func Post(path string, handler interface{}, annotation Annotation) {
	Register("POST", path, handler, annotation)
}

// Put is a convenience function for PUT routes
func Put(path string, handler interface{}, annotation Annotation) {
	Register("PUT", path, handler, annotation)
}

// Delete is a convenience function for DELETE routes
func Delete(path string, handler interface{}, annotation Annotation) {
	Register("DELETE", path, handler, annotation)
}

// Patch is a convenience function for PATCH routes
func Patch(path string, handler interface{}, annotation Annotation) {
	Register("PATCH", path, handler, annotation)
}

// Options is a convenience function for OPTIONS routes
func Options(path string, handler interface{}, annotation Annotation) {
	Register("OPTIONS", path, handler, annotation)
}

// Head is a convenience function for HEAD routes
func Head(path string, handler interface{}, annotation Annotation) {
	Register("HEAD", path, handler, annotation)
}

// Middleware returns middleware for serving OpenAPI documentation
func Middleware(g *Generator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/openapi.json" {
				json, err := g.ToJSON()
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write(json)
				return
			}

			if r.URL.Path == "/docs" || r.URL.Path == "/docs/" {
				// Serve Swagger UI
				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte(g.generateSwaggerUI()))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SwaggerUI returns a handler for serving Swagger UI
func (g *Generator) SwaggerUI(basePath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(g.generateSwaggerUI()))
	})
}

// OpenAPIJSON returns a handler for serving OpenAPI JSON
func (g *Generator) OpenAPIJSON(basePath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		jsonData, err := g.ToJSON()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(jsonData)
	})
}

// generateSwaggerUI generates HTML for Swagger UI
func (g *Generator) generateSwaggerUI() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui.css">
    <style>
        html { box-sizing: border-box; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin: 0; background: #fafafa; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: "/openapi.json",
                dom_id: "#swagger-ui",
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
            });
        };
    </script>
</body>
</html>`
}