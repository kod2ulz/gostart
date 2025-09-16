package docs

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/kod2ulz/gostart/contracts"
)

// OpenAPI represents the OpenAPI 3.0 specification structure
type OpenAPI struct {
	OpenAPI    string                 `json:"openapi"`
	Info       Info                   `json:"info"`
	Servers    []Server               `json:"servers"`
	Paths      map[string]interface{} `json:"paths"`
	Components Components             `json:"components"`
}

// Info contains API metadata
type Info struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// Server represents an API server
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

// Components contains reusable components
type Components struct {
	Schemas map[string]interface{} `json:"schemas"`
}

// PathItem represents an OpenAPI path item
type PathItem struct {
	Get     *Operation `json:"get,omitempty"`
	Post    *Operation `json:"post,omitempty"`
	Put     *Operation `json:"put,omitempty"`
	Delete  *Operation `json:"delete,omitempty"`
	Patch   *Operation `json:"patch,omitempty"`
	Options *Operation `json:"options,omitempty"`
	Head    *Operation `json:"head,omitempty"`
}

// Operation represents an API operation
type Operation struct {
	Summary     string                 `json:"summary,omitempty"`
	Description string                 `json:"description,omitempty"`
	OperationID string                 `json:"operationId,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Parameters  []Parameter            `json:"parameters,omitempty"`
	RequestBody *RequestBody           `json:"requestBody,omitempty"`
	Responses   map[string]interface{} `json:"responses"`
}

// Parameter represents an operation parameter
type Parameter struct {
	Name        string `json:"name"`
	In          string `json:"in"`          // "query", "header", "path", "cookie"
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
	Schema      Schema `json:"schema"`
}

// RequestBody represents a request body
type RequestBody struct {
	Description string                 `json:"description,omitempty"`
	Required    bool                   `json:"required"`
	Content     map[string]interface{} `json:"content"`
}

// Response represents an operation response
type Response struct {
	Description string                 `json:"description,omitempty"`
	Content     map[string]interface{} `json:"content,omitempty"`
}

// Schema represents a JSON schema
type Schema struct {
	Type                 string                 `json:"type,omitempty"`
	Description          string                 `json:"description,omitempty"`
	Format               string                 `json:"format,omitempty"`
	Enum                 []interface{}          `json:"enum,omitempty"`
	Items                *Schema                `json:"items,omitempty"`
	Properties           map[string]interface{} `json:"properties,omitempty"`
	Required             []string               `json:"required,omitempty"`
	AdditionalProperties bool                   `json:"additionalProperties,omitempty"`
	Ref                  string                 `json:"$ref,omitempty"`
}

// Generator generates OpenAPI documentation from request/response types
type Generator struct {
	baseURL    string
	title      string
	version    string
	schemas    map[string]interface{}
	operations map[string]*Operation
}

// NewGenerator creates a new documentation generator
func NewGenerator(baseURL, title, version string) *Generator {
	return &Generator{
		baseURL:    baseURL,
		title:      title,
		version:    version,
		schemas:    make(map[string]interface{}),
		operations: make(map[string]*Operation),
	}
}

// RegisterRoute registers a route for documentation generation
func (g *Generator) RegisterRoute(method, path string, handler interface{}, requestType, responseType reflect.Type) error {
	operation := g.generateOperation(method, path, handler, requestType, responseType)

	// Normalize path (replace path parameters with OpenAPI format)
	openAPIPath := convertPathToOpenAPI(path)

	// Add to paths
	if g.operations == nil {
		g.operations = make(map[string]*Operation)
	}
	g.operations[method+" "+openAPIPath] = operation

	return nil
}

// Generate generates the complete OpenAPI specification
func (g *Generator) Generate() (*OpenAPI, error) {
	// Build paths
	paths := make(map[string]interface{})

	for key, operation := range g.operations {
		parts := strings.SplitN(key, " ", 2)
		if len(parts) != 2 {
			continue
		}
		method, path := parts[0], parts[1]

		// Get or create path item
		var pathItem *PathItem
		if existing, exists := paths[path]; exists {
			if pi, ok := existing.(*PathItem); ok {
				pathItem = pi
			} else {
				// Convert existing to PathItem
				pathItem = &PathItem{}
			}
		} else {
			pathItem = &PathItem{}
		}

		// Add operation based on method
		switch strings.ToUpper(method) {
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

		paths[path] = pathItem
	}

	return &OpenAPI{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:       g.title,
			Description: "Automatically generated API documentation",
			Version:     g.version,
		},
		Servers: []Server{
			{
				URL:         g.baseURL,
				Description: "Development server",
			},
		},
		Paths: paths,
		Components: Components{
			Schemas: g.schemas,
		},
	}, nil
}

// ToJSON converts the OpenAPI specification to JSON
func (g *Generator) ToJSON() ([]byte, error) {
	spec, err := g.Generate()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(spec, "", "  ")
}

// generateOperation generates an OpenAPI operation from handler and types
func (g *Generator) generateOperation(method, path string, handler interface{}, requestType, responseType reflect.Type) *Operation {
	operation := &Operation{
		OperationID: fmt.Sprintf("%s%s", strings.ToLower(method), strings.ReplaceAll(path, "/", "_")),
		Responses: map[string]interface{}{
			"200": Response{
				Description: "Successful response",
			},
		},
	}

	// Analyze request type if provided
	if requestType != nil && requestType != reflect.TypeOf((*contracts.RequestParam)(nil)).Elem() {
		g.analyzeRequestType(operation, requestType)
	}

	// Analyze response type if provided
	if responseType != nil {
		g.analyzeResponseType(operation, responseType)
	}

	return operation
}

// analyzeRequestType analyzes a request type and generates OpenAPI parameters and schema
func (g *Generator) analyzeRequestType(operation *Operation, t reflect.Type) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return
	}

	// Generate schema for request type
	schemaName := t.Name()
	schema := g.generateSchema(t)
	g.schemas[schemaName] = schema

	// Analyze struct fields for parameters
	var bodyFields []reflect.StructField
	var hasJSONFields bool

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Check for parameter source tags
		if queryTag := field.Tag.Get("query"); queryTag != "" {
			operation.Parameters = append(operation.Parameters, Parameter{
				Name:     queryTag,
				In:       "query",
				Required: !isOptional(field),
				Schema:   generateFieldSchema(field),
			})
		} else if paramTag := field.Tag.Get("param"); paramTag != "" {
			operation.Parameters = append(operation.Parameters, Parameter{
				Name:     paramTag,
				In:       "path",
				Required: true, // Path parameters are always required
				Schema:   generateFieldSchema(field),
			})
		} else if headerTag := field.Tag.Get("header"); headerTag != "" {
			operation.Parameters = append(operation.Parameters, Parameter{
				Name:     headerTag,
				In:       "header",
				Required: !isOptional(field),
				Schema:   generateFieldSchema(field),
			})
		} else if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
			bodyFields = append(bodyFields, field)
			hasJSONFields = true
		}
	}

	// Create request body if there are JSON fields
	if hasJSONFields {
		operation.RequestBody = &RequestBody{
			Required: true,
			Content: map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": Schema{
						Ref: "#/components/schemas/" + schemaName,
					},
				},
			},
		}
	}
}

// analyzeResponseType analyzes a response type and generates OpenAPI schema
func (g *Generator) analyzeResponseType(operation *Operation, t reflect.Type) {
	// Handle response wrapper types
	if t.Name() == "Response" || strings.HasSuffix(t.Name(), "Response") {
		// Try to extract the inner type
		if t.NumField() > 0 {
			innerType := t.Field(0).Type
			if innerType.Kind() == reflect.Ptr {
				innerType = innerType.Elem()
			}

			if innerType.Kind() == reflect.Struct {
				schemaName := innerType.Name()
				schema := g.generateSchema(innerType)
				g.schemas[schemaName] = schema

				// Update response
				operation.Responses["200"] = Response{
					Description: "Successful response",
					Content: map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": Schema{
								Ref: "#/components/schemas/" + schemaName,
							},
						},
					},
				}
			}
		}
	}
}

// generateSchema generates a JSON schema from a Go type
func (g *Generator) generateSchema(t reflect.Type) Schema {
	schema := Schema{}

	switch t.Kind() {
	case reflect.String:
		schema.Type = "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema.Type = "integer"
		schema.Format = "int64"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema.Type = "integer"
		schema.Format = "int64"
	case reflect.Float32, reflect.Float64:
		schema.Type = "number"
		schema.Format = "double"
	case reflect.Bool:
		schema.Type = "boolean"
	case reflect.Slice:
		schema.Type = "array"
		itemType := t.Elem()
		itemSchema := g.generateSchema(itemType)
		schema.Items = &itemSchema
	case reflect.Struct:
		schema.Type = "object"
		properties := make(map[string]interface{})
		required := make([]string, 0)

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}

			jsonTag := field.Tag.Get("json")
			if jsonTag == "" || jsonTag == "-" {
				continue
			}

			fieldName := strings.Split(jsonTag, ",")[0]
			if fieldName == "" {
				fieldName = field.Name
			}

			properties[fieldName] = generateFieldSchema(field)

			if !isOptional(field) {
				required = append(required, fieldName)
			}
		}

		if len(properties) > 0 {
			schema.Properties = properties
		}
		if len(required) > 0 {
			schema.Required = required
		}
	}

	return schema
}

// Helper functions

func convertPathToOpenAPI(path string) string {
	// Convert path parameters from :param to {param}
	return strings.ReplaceAll(path, ":", "{")
}

func generateFieldSchema(field reflect.StructField) Schema {
	schema := generateBasicSchema(field.Type)

	// Add description from tag if present
	if desc := field.Tag.Get("description"); desc != "" {
		schema.Description = desc
	}

	// Handle enum values
	if enumTag := field.Tag.Get("enum"); enumTag != "" {
		// Simple enum parsing - could be enhanced
		schema.Enum = []interface{}{enumTag}
	}

	return schema
}

func generateBasicSchema(t reflect.Type) Schema {
	switch t.Kind() {
	case reflect.String:
		return Schema{Type: "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return Schema{Type: "integer", Format: "int64"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return Schema{Type: "integer", Format: "int64"}
	case reflect.Float32:
		return Schema{Type: "number", Format: "float"}
	case reflect.Float64:
		return Schema{Type: "number", Format: "double"}
	case reflect.Bool:
		return Schema{Type: "boolean"}
	default:
		return Schema{Type: "object"}
	}
}

func isOptional(field reflect.StructField) bool {
	// Check if field has optional tag or is a pointer
	optionalTag := field.Tag.Get("optional")
	return optionalTag == "true" || field.Type.Kind() == reflect.Ptr
}