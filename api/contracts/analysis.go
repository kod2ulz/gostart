package contracts

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// RequestContract represents the complete request specification
type RequestContract struct {
	PathParameters  map[string]ParameterContract `json:"pathParameters,omitempty"`
	QueryParameters map[string]ParameterContract `json:"queryParameters,omitempty"`
	Headers         map[string]ParameterContract `json:"headers,omitempty"`
	Body            *BodyContract                `json:"body,omitempty"`
	Cookies         map[string]ParameterContract `json:"cookies,omitempty"`
	ContentType     string                       `json:"contentType,omitempty"`
	Accept          string                       `json:"accept,omitempty"`
	Authorization   *AuthorizationContract       `json:"authorization,omitempty"`
}

// ResponseContract represents the complete response specification
type ResponseContract struct {
	StatusCode      int                          `json:"statusCode"`
	ContentType     string                       `json:"contentType,omitempty"`
	Headers         map[string]ParameterContract `json:"headers,omitempty"`
	Body            *BodyContract                `json:"body,omitempty"`
	File            *FileContract                `json:"file,omitempty"`
	Stream          *StreamContract              `json:"stream,omitempty"`
	ErrorResponses  map[int]ResponseContract     `json:"errorResponses,omitempty"`
	IsPartial       bool                         `json:"isPartial,omitempty"`
	HiddenFields    []string                     `json:"hiddenFields,omitempty"`
	SanitizedFields []string                     `json:"sanitizedFields,omitempty"`
}

// ParameterContract represents a parameter specification
type ParameterContract struct {
	Name        string        `json:"name"`
	Type        string        `json:"type"`
	Format      string        `json:"format,omitempty"`
	In          string        `json:"in,omitempty"`
	Required    bool          `json:"required"`
	Default     interface{}   `json:"default,omitempty"`
	Enum        []interface{} `json:"enum,omitempty"`
	Min         *float64      `json:"min,omitempty"`
	Max         *float64      `json:"max,omitempty"`
	MinLength   *int          `json:"minLength,omitempty"`
	MaxLength   *int          `json:"maxLength,omitempty"`
	Pattern     string        `json:"pattern,omitempty"`
	Description string        `json:"description,omitempty"`
	Example     interface{}   `json:"example,omitempty"`
	Deprecated  bool          `json:"deprecated,omitempty"`
	ReadOnly    bool          `json:"readOnly,omitempty"`
	WriteOnly   bool          `json:"writeOnly,omitempty"`
}

// BodyContract represents a request/response body specification
type BodyContract struct {
	ContentType string              `json:"contentType"`
	Schema      *SchemaContract     `json:"schema,omitempty"`
	Example     interface{}         `json:"example,omitempty"`
	Encoding    map[string]Encoding `json:"encoding,omitempty"`
	Required    bool                `json:"required"`
}

// SchemaContract represents a JSON schema specification
type SchemaContract struct {
	Type                 string                    `json:"type,omitempty"`
	Format               string                    `json:"format,omitempty"`
	Description          string                    `json:"description,omitempty"`
	Properties           map[string]SchemaContract `json:"properties,omitempty"`
	Required             []string                  `json:"required,omitempty"`
	Items                *SchemaContract           `json:"items,omitempty"`
	AdditionalProperties bool                      `json:"additionalProperties,omitempty"`
	Enum                 []interface{}             `json:"enum,omitempty"`
	Default              interface{}               `json:"default,omitempty"`
	Example              interface{}               `json:"example,omitempty"`
	MinLength            *int                      `json:"minLength,omitempty"`
	MaxLength            *int                      `json:"maxLength,omitempty"`
	Pattern              string                    `json:"pattern,omitempty"`
	Minimum              *float64                  `json:"minimum,omitempty"`
	Maximum              *float64                  `json:"maximum,omitempty"`
	ExclusiveMinimum     bool                      `json:"exclusiveMinimum,omitempty"`
	ExclusiveMaximum     bool                      `json:"exclusiveMaximum,omitempty"`
	MultipleOf           *float64                  `json:"multipleOf,omitempty"`
	MinItems             *int                      `json:"minItems,omitempty"`
	MaxItems             *int                      `json:"maxItems,omitempty"`
	UniqueItems          bool                      `json:"uniqueItems,omitempty"`
	MinProperties        *int                      `json:"minProperties,omitempty"`
	MaxProperties        *int                      `json:"maxProperties,omitempty"`
	AllOf                []SchemaContract          `json:"allOf,omitempty"`
	AnyOf                []SchemaContract          `json:"anyOf,omitempty"`
	OneOf                []SchemaContract          `json:"oneOf,omitempty"`
	Not                  *SchemaContract           `json:"not,omitempty"`
	Ref                  string                    `json:"$ref,omitempty"`
	ReadOnly             bool                      `json:"readOnly,omitempty"`
	WriteOnly            bool                      `json:"writeOnly,omitempty"`
	Deprecated           bool                      `json:"deprecated,omitempty"`
}

// Encoding represents encoding information for multipart requests
type Encoding struct {
	ContentType   string                       `json:"contentType,omitempty"`
	Headers       map[string]ParameterContract `json:"headers,omitempty"`
	Style         string                       `json:"style,omitempty"`
	Explode       bool                         `json:"explode,omitempty"`
	AllowReserved bool                         `json:"allowReserved,omitempty"`
}

// FileContract represents a file response specification
type FileContract struct {
	ContentType   string `json:"contentType,omitempty"`
	Filename      string `json:"filename,omitempty"`
	ContentLength *int64 `json:"contentLength,omitempty"`
	ETag          string `json:"etag,omitempty"`
	LastModified  string `json:"lastModified,omitempty"`
	Disposition   string `json:"disposition,omitempty"`
}

// StreamContract represents a streaming response specification
type StreamContract struct {
	ContentType string `json:"contentType,omitempty"`
	Chunked     bool   `json:"chunked,omitempty"`
	EventStream bool   `json:"eventStream,omitempty"`
}

// AuthorizationContract represents authorization requirements
type AuthorizationContract struct {
	Type         string   `json:"type"` // "bearer", "basic", "apiKey", "oauth2"
	Scheme       string   `json:"scheme,omitempty"`
	BearerFormat string   `json:"bearerFormat,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
	In           string   `json:"in,omitempty"` // "header", "query", "cookie"
	Name         string   `json:"name,omitempty"`
}

// ContractAnalyzer analyzes request/response contracts from handlers
type ContractAnalyzer struct {
	typeCache      map[reflect.Type]*SchemaContract
	knownTypes     map[string]reflect.Type
	customMappings map[string]func(reflect.Type) *SchemaContract
}

// NewContractAnalyzer creates a new contract analyzer
func NewContractAnalyzer() *ContractAnalyzer {
	return &ContractAnalyzer{
		typeCache:      make(map[reflect.Type]*SchemaContract),
		knownTypes:     make(map[string]reflect.Type),
		customMappings: make(map[string]func(reflect.Type) *SchemaContract),
	}
}

// AnalyzeRequest analyzes a request contract from handler function and route path
func (ca *ContractAnalyzer) AnalyzeRequest(handler interface{}, path string) (*RequestContract, error) {
	contract := &RequestContract{
		PathParameters:  make(map[string]ParameterContract),
		QueryParameters: make(map[string]ParameterContract),
		Headers:         make(map[string]ParameterContract),
		Cookies:         make(map[string]ParameterContract),
	}

	// Extract path parameters from route path
	ca.extractPathParameters(path, contract)

	// Analyze handler function to extract request parameters
	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() == reflect.Func {
		ca.analyzeHandlerFunction(handlerType, contract, path)
	}

	// Set default content type and accept headers
	if contract.ContentType == "" {
		contract.ContentType = "application/json"
	}
	if contract.Accept == "" {
		contract.Accept = "application/json"
	}

	return contract, nil
}

// AnalyzeResponse analyzes a response contract from handler function
func (ca *ContractAnalyzer) AnalyzeResponse(handler interface{}) (*ResponseContract, error) {
	contract := &ResponseContract{
		StatusCode:     http.StatusOK,
		Headers:        make(map[string]ParameterContract),
		ErrorResponses: make(map[int]ResponseContract),
	}

	// Analyze handler function to extract response information
	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() == reflect.Func {
		ca.analyzeHandlerResponse(handlerType, contract)
	}

	// Set default content type
	if contract.ContentType == "" {
		contract.ContentType = "application/json"
	}

	return contract, nil
}

// extractPathParameters extracts path parameters from route path
func (ca *ContractAnalyzer) extractPathParameters(path string, contract *RequestContract) {
	// Extract path parameters using regex
	pathParamRegex := regexp.MustCompile(`:(\w+)`)
	matches := pathParamRegex.FindAllStringSubmatch(path, -1)

	for _, match := range matches {
		if len(match) > 1 {
			paramName := match[1]
			contract.PathParameters[paramName] = ParameterContract{
				Name:     paramName,
				Type:     "string",
				Required: true,
				In:       "path",
			}
		}
	}
}

// analyzeHandlerFunction analyzes handler function to extract request parameters
func (ca *ContractAnalyzer) analyzeHandlerFunction(handlerType reflect.Type, contract *RequestContract, path string) {
	// Handler function should have one parameter: RequestContext
	if handlerType.NumIn() != 1 {
		return
	}

	contextType := handlerType.In(0)
	if contextType.Name() != "RequestContext" {
		return
	}

	// Analyze handler function name and route path to infer operation type
	handlerName := ca.getHandlerName(handlerType)
	ca.analyzeOperationPattern(handlerName, path, contract)

	// Don't add generic context method parameters - they should be explicitly defined
	// ca.analyzeContextMethods(contextType, contract)
}

// getHandlerName attempts to extract the handler function name
func (ca *ContractAnalyzer) getHandlerName(handlerType reflect.Type) string {
	// This is a simplified approach - in a real implementation,
	// you would need to get the actual function name from the handler
	if handlerType.Kind() == reflect.Func {
		return "handler"
	}
	return handlerType.Name()
}

// analyzeOperationPattern analyzes the handler name and route path to infer operation patterns
func (ca *ContractAnalyzer) analyzeOperationPattern(handlerName, routePath string, contract *RequestContract) {
	// Only add body if the operation typically needs one (POST, PUT, PATCH)
	// GET, DELETE, HEAD, OPTIONS typically don't have bodies
	lowerName := strings.ToLower(handlerName)
	needsBody := strings.Contains(lowerName, "create") || strings.Contains(lowerName, "add") ||
		strings.Contains(lowerName, "update") || strings.Contains(lowerName, "edit") ||
		strings.Contains(lowerName, "patch")

	if needsBody {
		// Initialize Body for operations that need it
		if contract.Body == nil {
			contract.Body = &BodyContract{
				ContentType: "application/json",
				Required:    true,
				Schema: &SchemaContract{
					Type:       "object",
					Properties: make(map[string]SchemaContract),
				},
			}
		}
	}

	// Analyze route path for pagination patterns
	if strings.HasSuffix(routePath, "/") || strings.Contains(routePath, "/list") {
		// List operation - add pagination parameters
		contract.QueryParameters["page"] = ParameterContract{
			Name:     "page",
			Type:     "integer",
			Required: false,
			In:       "query",
			Default:  1,
		}
		contract.QueryParameters["pageSize"] = ParameterContract{
			Name:     "pageSize",
			Type:     "integer",
			Required: false,
			In:       "query",
			Default:  20,
		}
		contract.QueryParameters["sortBy"] = ParameterContract{
			Name:     "sortBy",
			Type:     "string",
			Required: false,
			In:       "query",
		}
		contract.QueryParameters["sortOrder"] = ParameterContract{
			Name:     "sortOrder",
			Type:     "string",
			Required: false,
			In:       "query",
			Enum:     []interface{}{"asc", "desc"},
		}
	}
}

// analyzeContextMethods analyzes RequestContext methods to determine parameter usage
func (ca *ContractAnalyzer) analyzeContextMethods(contextType reflect.Type, contract *RequestContract) {
	// Common query parameters for list operations
	contract.QueryParameters["filter"] = ParameterContract{
		Name:     "filter",
		Type:     "string",
		Required: false,
		In:       "query",
	}
	contract.QueryParameters["limit"] = ParameterContract{
		Name:     "limit",
		Type:     "integer",
		Required: false,
		In:       "query",
	}
	contract.QueryParameters["offset"] = ParameterContract{
		Name:     "offset",
		Type:     "integer",
		Required: false,
		In:       "query",
	}

	// Common headers
	contract.Headers["Authorization"] = ParameterContract{
		Name:     "Authorization",
		Type:     "string",
		Required: false,
		In:       "header",
	}
	contract.Headers["Content-Type"] = ParameterContract{
		Name:     "Content-Type",
		Type:     "string",
		Required: false,
		In:       "header",
		Default:  "application/json",
	}

	// Default JSON body
	contract.ContentType = "application/json"
	contract.Body = &BodyContract{
		ContentType: "application/json",
		Required:    true,
		Schema: &SchemaContract{
			Type: "object",
			Properties: map[string]SchemaContract{
				"data": {
					Type: "object",
				},
			},
		},
	}

	// Cookies
	contract.Cookies["session"] = ParameterContract{
		Name:     "session",
		Type:     "string",
		Required: false,
		In:       "cookie",
	}
}

// analyzeHandlerResponse analyzes handler function to extract response information
func (ca *ContractAnalyzer) analyzeHandlerResponse(handlerType reflect.Type, contract *ResponseContract) {
	// Default response structure
	contract.ContentType = "application/json"
	contract.Body = &BodyContract{
		ContentType: "application/json",
		Required:    true,
		Schema: &SchemaContract{
			Type: "object",
			Properties: map[string]SchemaContract{
				"success": {
					Type:    "boolean",
					Default: true,
				},
				"data": {
					Type: "object",
				},
			},
		},
	}

	// Analyze return types to extract more detailed response information
	if handlerType.NumOut() > 0 {
		ca.analyzeReturnTypes(handlerType, contract)
	}

	// Check for error handling
	ca.analyzeErrorHandling(handlerType, contract)

	// Add common response headers
	contract.Headers["Content-Type"] = ParameterContract{
		Name:     "Content-Type",
		Type:     "string",
		Required: true,
		In:       "header",
		Default:  "application/json",
	}
	contract.Headers["X-Request-ID"] = ParameterContract{
		Name:        "X-Request-ID",
		Type:        "string",
		Required:    false,
		In:          "header",
		Description: "Unique request identifier",
	}
}

// analyzeReturnTypes analyzes return types to extract response structure
func (ca *ContractAnalyzer) analyzeReturnTypes(handlerType reflect.Type, contract *ResponseContract) {
	// Look at return types to determine response structure
	for i := 0; i < handlerType.NumOut(); i++ {
		returnType := handlerType.Out(i)

		// Skip error types
		if returnType.String() == "ierrors.Error" || returnType.String() == "error" {
			continue
		}

		// Analyze the return type to create response schema
		responseSchema := ca.AnalyzeType(returnType)

		// Handle different response patterns based on return type
		ca.handleResponseTypePattern(returnType, responseSchema, contract)
	}
}

// handleResponseTypePattern handles different response type patterns
func (ca *ContractAnalyzer) handleResponseTypePattern(returnType reflect.Type, responseSchema *SchemaContract, contract *ResponseContract) {
	if contract.Body == nil || contract.Body.Schema == nil {
		return
	}

	// Handle array responses (list operations)
	if returnType.Kind() == reflect.Slice || returnType.Kind() == reflect.Array {
		// List response - add pagination metadata
		contract.Body.Schema.Properties = map[string]SchemaContract{
			"success": {
				Type:    "boolean",
				Default: true,
			},
			"data": {
				Type:  "array",
				Items: responseSchema,
			},
			"pagination": {
				Type: "object",
				Properties: map[string]SchemaContract{
					"page": {
						Type:    "integer",
						Default: 1,
					},
					"pageSize": {
						Type:    "integer",
						Default: 20,
					},
					"total": {
						Type: "integer",
					},
					"totalPages": {
						Type: "integer",
					},
				},
			},
		}
		return
	}

	// Handle map responses (status responses, metadata)
	if returnType.Kind() == reflect.Map {
		// Map response - typically used for status or simple responses
		contract.Body.Schema.Properties = map[string]SchemaContract{
			"success": {
				Type:    "boolean",
				Default: true,
			},
			"data": *responseSchema,
		}
		return
	}

	// Handle pointer responses (single resource)
	if returnType.Kind() == reflect.Ptr {
		// Single resource response
		contract.Body.Schema.Properties = map[string]SchemaContract{
			"success": {
				Type:    "boolean",
				Default: true,
			},
			"data": *responseSchema,
		}
		return
	}

	// Handle primitive types (simple responses)
	if ca.isPrimitiveType(returnType) {
		// Primitive response - often used for simple values or counts
		contract.Body.Schema.Properties = map[string]SchemaContract{
			"success": {
				Type:    "boolean",
				Default: true,
			},
			"data": *responseSchema,
		}
		return
	}

	// Default - struct response
	contract.Body.Schema.Properties["data"] = *responseSchema
}

// isPrimitiveType checks if a type is a primitive Go type
func (ca *ContractAnalyzer) isPrimitiveType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Bool:
		return true
	}
	return false
}

// extractStructFields extracts field information from struct types
func (ca *ContractAnalyzer) extractStructFields(t reflect.Type) map[string]SchemaContract {
	if t.Kind() != reflect.Struct {
		return nil
	}

	fields := make(map[string]SchemaContract)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		// Get field name from JSON tag or field name
		fieldName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			if jsonTag == "-" {
				continue // Skip explicitly ignored fields
			}
			fieldName = strings.Split(jsonTag, ",")[0]
		}

		// Analyze field type
		fieldSchema := ca.AnalyzeType(field.Type)

		// Add validation information
		if validateTag := field.Tag.Get("validate"); validateTag != "" {
			ca.applyValidationRules(fieldSchema, validateTag)
		}

		// Add example information
		if exampleTag := field.Tag.Get("example"); exampleTag != "" {
			fieldSchema.Example = exampleTag
		}

		// Add default value
		if defaultTag := field.Tag.Get("default"); defaultTag != "" {
			fieldSchema.Default = defaultTag
		}

		// Check if field is read-only
		if readonlyTag := field.Tag.Get("readonly"); readonlyTag == "true" {
			fieldSchema.ReadOnly = true
		}

		// Add description from comment (would require AST parsing in real implementation)
		fieldSchema.Description = fieldName

		fields[fieldName] = *fieldSchema
	}

	return fields
}

// applyValidationRules applies validation rules from struct tags
func (ca *ContractAnalyzer) applyValidationRules(schema *SchemaContract, validateTag string) {
	rules := strings.Split(validateTag, ",")

	for _, rule := range rules {
		if rule == "" {
			continue
		}

		switch {
		case rule == "required":
			// This would be handled at the struct level
		case strings.HasPrefix(rule, "min="):
			if minStr := strings.TrimPrefix(rule, "min="); minStr != "" {
				if min, err := strconv.Atoi(minStr); err == nil {
					schema.MinLength = &min
					minFloat := float64(min)
					schema.Minimum = &minFloat
				}
			}
		case strings.HasPrefix(rule, "max="):
			if maxStr := strings.TrimPrefix(rule, "max="); maxStr != "" {
				if max, err := strconv.Atoi(maxStr); err == nil {
					schema.MaxLength = &max
					maxFloat := float64(max)
					schema.Maximum = &maxFloat
				}
			}
		case rule == "email":
			schema.Format = "email"
		case rule == "url":
			schema.Format = "uri"
		case rule == "uuid":
			schema.Format = "uuid"
		case strings.HasPrefix(rule, "gte="):
			if gteStr := strings.TrimPrefix(rule, "gte="); gteStr != "" {
				if gte, err := strconv.Atoi(gteStr); err == nil {
					gteFloat := float64(gte)
					schema.Minimum = &gteFloat
				}
			}
		case strings.HasPrefix(rule, "lte="):
			if lteStr := strings.TrimPrefix(rule, "lte="); lteStr != "" {
				if lte, err := strconv.Atoi(lteStr); err == nil {
					lteFloat := float64(lte)
					schema.Maximum = &lteFloat
				}
			}
		}
	}
}

// analyzeErrorHandling analyzes error handling in the handler function
func (ca *ContractAnalyzer) analyzeErrorHandling(handlerType reflect.Type, contract *ResponseContract) {
	if contract.ErrorResponses == nil {
		contract.ErrorResponses = make(map[int]ResponseContract)
	}

	// Check for error return types
	for i := 0; i < handlerType.NumOut(); i++ {
		returnType := handlerType.Out(i)
		if returnType.String() == "ierrors.Error" || returnType.String() == "error" {
			// Add error response schemas
			contract.ErrorResponses[400] = ResponseContract{
				StatusCode:  400,
				ContentType: "application/json",
				Body: &BodyContract{
					ContentType: "application/json",
					Schema: &SchemaContract{
						Type: "object",
						Properties: map[string]SchemaContract{
							"success": {
								Type:    "boolean",
								Default: false,
							},
							"error": {
								Type: "object",
								Properties: map[string]SchemaContract{
									"code": {
										Type: "string",
									},
									"message": {
										Type: "string",
									},
									"fields": {
										Type: "object",
									},
								},
							},
						},
					},
				},
			}

			contract.ErrorResponses[401] = ResponseContract{
				StatusCode:  401,
				ContentType: "application/json",
				Body: &BodyContract{
					ContentType: "application/json",
					Schema: &SchemaContract{
						Type: "object",
						Properties: map[string]SchemaContract{
							"success": {
								Type:    "boolean",
								Default: false,
							},
							"error": {
								Type: "object",
								Properties: map[string]SchemaContract{
									"code": {
										Type: "string",
									},
									"message": {
										Type: "string",
									},
								},
							},
						},
					},
				},
			}

			contract.ErrorResponses[404] = ResponseContract{
				StatusCode:  404,
				ContentType: "application/json",
				Body: &BodyContract{
					ContentType: "application/json",
					Schema: &SchemaContract{
						Type: "object",
						Properties: map[string]SchemaContract{
							"success": {
								Type:    "boolean",
								Default: false,
							},
							"error": {
								Type: "object",
								Properties: map[string]SchemaContract{
									"code": {
										Type: "string",
									},
									"message": {
										Type: "string",
									},
								},
							},
						},
					},
				},
			}

			contract.ErrorResponses[500] = ResponseContract{
				StatusCode:  500,
				ContentType: "application/json",
				Body: &BodyContract{
					ContentType: "application/json",
					Schema: &SchemaContract{
						Type: "object",
						Properties: map[string]SchemaContract{
							"success": {
								Type:    "boolean",
								Default: false,
							},
							"error": {
								Type: "object",
								Properties: map[string]SchemaContract{
									"code": {
										Type: "string",
									},
									"message": {
										Type: "string",
									},
								},
							},
						},
					},
				},
			}
		}
	}
}

// AnalyzeType analyzes a Go type and returns its schema contract
func (ca *ContractAnalyzer) AnalyzeType(t reflect.Type) *SchemaContract {
	if schema, cached := ca.typeCache[t]; cached {
		return schema
	}

	schema := ca.analyzeTypeRecursive(t)
	ca.typeCache[t] = schema
	return schema
}

// analyzeTypeRecursive recursively analyzes a type
func (ca *ContractAnalyzer) analyzeTypeRecursive(t reflect.Type) *SchemaContract {
	schema := &SchemaContract{}

	switch t.Kind() {
	case reflect.String:
		schema.Type = "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema.Type = "integer"
		if t.Kind() == reflect.Int64 {
			schema.Format = "int64"
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema.Type = "integer"
		schema.Format = "int64"
	case reflect.Float32, reflect.Float64:
		schema.Type = "number"
		if t.Kind() == reflect.Float32 {
			schema.Format = "float"
		} else {
			schema.Format = "double"
		}
	case reflect.Bool:
		schema.Type = "boolean"
	case reflect.Slice, reflect.Array:
		schema.Type = "array"
		if t.Elem() != nil {
			schema.Items = ca.analyzeTypeRecursive(t.Elem())
		}
	case reflect.Map:
		schema.Type = "object"
		if t.Key() != nil && t.Key().Kind() == reflect.String {
			schema.AdditionalProperties = true
		}
	case reflect.Struct:
		schema.Type = "object"
		schema.Properties = ca.extractStructFields(t)
		schema.Required = make([]string, 0)

		// Extract required fields from json tags and validation tags
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" && !field.Anonymous {
				// Skip unexported fields
				continue
			}

			fieldName := field.Name
			if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
				parts := strings.Split(tag, ",")
				if parts[0] != "" {
					fieldName = parts[0]
				}
			}

			// Check if field is required
			isRequired := false

			// Check json tag for omitempty
			if jsonTag := field.Tag.Get("json"); jsonTag != "" && !strings.Contains(jsonTag, "omitempty") {
				isRequired = true
			}

			// Check validation tag for required
			if validateTag := field.Tag.Get("validate"); validateTag != "" && strings.Contains(validateTag, "required") {
				isRequired = true
			}

			if isRequired && !slices.Contains(schema.Required, fieldName) {
				schema.Required = append(schema.Required, fieldName)
			}
		}

		if len(schema.Required) == 0 {
			schema.Required = nil
		}

	case reflect.Ptr:
		if t.Elem() != nil {
			schema = ca.analyzeTypeRecursive(t.Elem())
		}
	case reflect.Interface:
		schema.Type = "object"
	default:
		schema.Type = "object"
	}

	return schema
}

// applyValidationTags applies validation tags to schema
func (ca *ContractAnalyzer) applyValidationTags(schema *SchemaContract, validateTag string) {
	tags := strings.Split(validateTag, ",")
	for _, tag := range tags {
		switch {
		case tag == "required":
			// This is handled by the json tag analysis
		case strings.HasPrefix(tag, "min="):
			if schema.Type == "string" {
				val := strings.TrimPrefix(tag, "min=")
				if minVal, err := parseInt(val); err == nil {
					schema.MinLength = &minVal
				}
			} else if schema.Type == "integer" || schema.Type == "number" {
				val := strings.TrimPrefix(tag, "min=")
				if minVal, err := parseFloat(val); err == nil {
					schema.Minimum = &minVal
				}
			}
		case strings.HasPrefix(tag, "max="):
			if schema.Type == "string" {
				val := strings.TrimPrefix(tag, "max=")
				if maxVal, err := parseInt(val); err == nil {
					schema.MaxLength = &maxVal
				}
			} else if schema.Type == "integer" || schema.Type == "number" {
				val := strings.TrimPrefix(tag, "max=")
				if maxVal, err := parseFloat(val); err == nil {
					schema.Maximum = &maxVal
				}
			}
		case strings.HasPrefix(tag, "email"):
			schema.Format = "email"
		case strings.HasPrefix(tag, "url"):
			schema.Format = "uri"
		case strings.HasPrefix(tag, "uuid"):
			schema.Format = "uuid"
		}
	}
}

// parseInt helper function
func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// parseFloat helper function
func parseFloat(s string) (float64, error) {
	var result float64
	_, err := fmt.Sscanf(s, "%f", &result)
	return result, err
}
