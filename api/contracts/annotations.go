package contracts

import (
	"reflect"
	"strings"
)

// Annotation represents metadata that can be attached to routes and handlers
// to customize OpenAPI documentation generation
type Annotation struct {
	// Basic Information
	Summary     string   `json:"summary,omitempty"`
	Description string   `json:"description,omitempty"`
	OperationID string  `json:"operationId,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Deprecated  bool     `json:"deprecated,omitempty"`

	// Request/Response Information
	Consumes []string `json:"consumes,omitempty"`
	Produces []string `json:"produces,omitempty"`

	// Parameters
	Parameters []ParameterAnnotation `json:"parameters,omitempty"`

	// Security
	Security []map[string][]string `json:"security,omitempty"`

	// Response Specifications
	Responses map[int]ResponseAnnotation `json:"responses,omitempty"`

	// External Documentation
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty"`

	// Custom Extensions
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// ParameterAnnotation represents a parameter annotation
type ParameterAnnotation struct {
	Name        string      `json:"name"`
	In          string      `json:"in"`          // "query", "header", "path", "cookie"
	Description string      `json:"description,omitempty"`
	Required    bool        `json:"required"`
	Schema      *Schema     `json:"schema,omitempty"`
	Example     interface{} `json:"example,omitempty"`
	Deprecated  bool        `json:"deprecated,omitempty"`
}

// ResponseAnnotation represents a response annotation
type ResponseAnnotation struct {
	Description string                 `json:"description"`
	Headers     map[string]Header      `json:"headers,omitempty"`
	Content     map[string]MediaType  `json:"content,omitempty"`
	Links       map[string]Link        `json:"links,omitempty"`
}

// ExternalDocs represents external documentation
type ExternalDocs struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`
}

// Schema represents a JSON schema
type Schema struct {
	Type                 string             `json:"type,omitempty"`
	Format               string             `json:"format,omitempty"`
	Description          string             `json:"description,omitempty"`
	Enum                 []interface{}      `json:"enum,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	Properties           map[string]Schema   `json:"properties,omitempty"`
	Required             []string           `json:"required,omitempty"`
	AllOf                []Schema           `json:"allOf,omitempty"`
	AnyOf                []Schema           `json:"anyOf,omitempty"`
	OneOf                []Schema           `json:"oneOf,omitempty"`
	Not                  *Schema            `json:"not,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty"`
	Default              interface{}        `json:"default,omitempty"`
	Example              interface{}        `json:"example,omitempty"`
	Ref                  string             `json:"$ref,omitempty"`
}

// Header represents a header definition
type Header struct {
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required,omitempty"`
	Deprecated  bool    `json:"deprecated,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
	Example     interface{} `json:"example,omitempty"`
}

// MediaType represents a media type
type MediaType struct {
	Schema   *Schema               `json:"schema,omitempty"`
	Example  interface{}           `json:"example,omitempty"`
	Examples map[string]Example    `json:"examples,omitempty"`
	Encoding map[string]MediaTypeEncoding `json:"encoding,omitempty"`
}

// Example represents an example
type Example struct {
	Summary     string      `json:"summary,omitempty"`
	Description string      `json:"description,omitempty"`
	Value       interface{} `json:"value,omitempty"`
	ExternalValue string    `json:"externalValue,omitempty"`
}

// MediaTypeEncoding represents encoding information
type MediaTypeEncoding struct {
	ContentType    string            `json:"contentType,omitempty"`
	Headers       map[string]Header `json:"headers,omitempty"`
	Style         string            `json:"style,omitempty"`
	Explode       bool              `json:"explode,omitempty"`
	AllowReserved bool              `json:"allowReserved,omitempty"`
}

// Link represents a link
type Link struct {
	OperationRef string            `json:"operationRef,omitempty"`
	OperationID  string            `json:"operationId,omitempty"`
	Description  string            `json:"description,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	RequestBody  interface{}        `json:"requestBody,omitempty"`
	Server       *Server           `json:"server,omitempty"`
}

// Server represents a server
type Server struct {
	URL         string                     `json:"url"`
	Description string                     `json:"description,omitempty"`
	Variables   map[string]ServerVariable  `json:"variables,omitempty"`
}

// ServerVariable represents a server variable
type ServerVariable struct {
	Enum        []string `json:"enum,omitempty"`
	Default     string   `json:"default"`
	Description string   `json:"description,omitempty"`
}

// Annotatable represents an interface for types that can be annotated
type Annotatable interface {
	GetAnnotations() []Annotation
	AddAnnotation(annotation Annotation)
}

// AnnotationStore manages annotations for routes and handlers
type AnnotationStore struct {
	routeAnnotations  map[string][]Annotation
	handlerAnnotations map[reflect.Type][]Annotation
}

// NewAnnotationStore creates a new annotation store
func NewAnnotationStore() *AnnotationStore {
	return &AnnotationStore{
		routeAnnotations:  make(map[string][]Annotation),
		handlerAnnotations: make(map[reflect.Type][]Annotation),
	}
}

// AddRouteAnnotation adds annotations for a specific route
func (as *AnnotationStore) AddRouteAnnotation(method, path string, annotation Annotation) {
	key := as.routeKey(method, path)
	as.routeAnnotations[key] = append(as.routeAnnotations[key], annotation)
}

// AddHandlerAnnotation adds annotations for a handler function
func (as *AnnotationStore) AddHandlerAnnotation(handler interface{}, annotation Annotation) {
	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() == reflect.Func {
		as.handlerAnnotations[handlerType] = append(as.handlerAnnotations[handlerType], annotation)
	}
}

// GetRouteAnnotations retrieves annotations for a route
func (as *AnnotationStore) GetRouteAnnotations(method, path string) []Annotation {
	key := as.routeKey(method, path)
	return as.routeAnnotations[key]
}

// GetHandlerAnnotations retrieves annotations for a handler
func (as *AnnotationStore) GetHandlerAnnotations(handler interface{}) []Annotation {
	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() == reflect.Func {
		return as.handlerAnnotations[handlerType]
	}
	return nil
}

// GetCombinedAnnotations retrieves combined annotations for route and handler
func (as *AnnotationStore) GetCombinedAnnotations(method, path string, handler interface{}) []Annotation {
	routeAnnotations := as.GetRouteAnnotations(method, path)
	handlerAnnotations := as.GetHandlerAnnotations(handler)

	// Merge annotations, with handler annotations taking precedence
	combined := make([]Annotation, len(routeAnnotations))
	copy(combined, routeAnnotations)

	// Merge with handler annotations
	for _, handlerAnnotation := range handlerAnnotations {
		merged := false
		for i, routeAnnotation := range combined {
			if canMergeAnnotations(routeAnnotation, handlerAnnotation) {
				combined[i] = mergeAnnotations(routeAnnotation, handlerAnnotation)
				merged = true
				break
			}
		}
		if !merged {
			combined = append(combined, handlerAnnotation)
		}
	}

	return combined
}

// routeKey creates a consistent key for route annotations
func (as *AnnotationStore) routeKey(method, path string) string {
	return strings.ToUpper(method) + ":" + path
}

// canMergeAnnotations determines if two annotations can be merged
func canMergeAnnotations(a1, a2 Annotation) bool {
	// Merge if they have the same operation ID or if one doesn't have one
	return a1.OperationID == a2.OperationID || a1.OperationID == "" || a2.OperationID == ""
}

// mergeAnnotations merges two annotations, with the second taking precedence
func mergeAnnotations(a1, a2 Annotation) Annotation {
	merged := a1

	// Override non-empty fields from a2
	if a2.Summary != "" {
		merged.Summary = a2.Summary
	}
	if a2.Description != "" {
		merged.Description = a2.Description
	}
	if a2.OperationID != "" {
		merged.OperationID = a2.OperationID
	}
	if a2.Deprecated {
		merged.Deprecated = a2.Deprecated
	}

	// Merge arrays
	merged.Tags = mergeStringArrays(merged.Tags, a2.Tags)
	merged.Consumes = mergeStringArrays(merged.Consumes, a2.Consumes)
	merged.Produces = mergeStringArrays(merged.Produces, a2.Produces)

	// Merge parameters (a2 takes precedence for parameters with same name)
	merged.Parameters = mergeParameters(merged.Parameters, a2.Parameters)

	// Merge security
	merged.Security = mergeSecurity(merged.Security, a2.Security)

	// Merge responses
	if merged.Responses == nil {
		merged.Responses = a2.Responses
	} else if a2.Responses != nil {
		for status, response := range a2.Responses {
			merged.Responses[status] = response
		}
	}

	// Merge external docs
	if a2.ExternalDocs != nil {
		merged.ExternalDocs = a2.ExternalDocs
	}

	// Merge extensions
	if merged.Extensions == nil {
		merged.Extensions = a2.Extensions
	} else if a2.Extensions != nil {
		for key, value := range a2.Extensions {
			merged.Extensions[key] = value
		}
	}

	return merged
}

// mergeStringArrays merges two string arrays, removing duplicates
func mergeStringArrays(a1, a2 []string) []string {
	result := make([]string, 0, len(a1)+len(a2))
	seen := make(map[string]bool)

	for _, item := range a1 {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	for _, item := range a2 {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// mergeParameters merges two parameter arrays, with second taking precedence
func mergeParameters(p1, p2 []ParameterAnnotation) []ParameterAnnotation {
	result := make([]ParameterAnnotation, 0, len(p1)+len(p2))
	nameMap := make(map[string]bool)

	// Add first array
	for _, param := range p1 {
		result = append(result, param)
		nameMap[param.Name] = true
	}

	// Add second array, overriding duplicates
	for _, param := range p2 {
		if nameMap[param.Name] {
			// Replace existing parameter
			for i, existing := range result {
				if existing.Name == param.Name {
					result[i] = param
					break
				}
			}
		} else {
			result = append(result, param)
			nameMap[param.Name] = true
		}
	}

	return result
}

// mergeSecurity merges security requirements
func mergeSecurity(s1, s2 []map[string][]string) []map[string][]string {
	if len(s2) == 0 {
		return s1
	}
	return s2 // s2 takes precedence
}

// Convenience functions for creating common annotations

// NewSummary creates an annotation with just a summary
func NewSummary(summary string) Annotation {
	return Annotation{Summary: summary}
}

// NewDescription creates an annotation with just a description
func NewDescription(description string) Annotation {
	return Annotation{Description: description}
}

// NewTag creates an annotation with tags
func NewTag(tags ...string) Annotation {
	return Annotation{Tags: tags}
}

// NewOperationID creates an annotation with an operation ID
func NewOperationID(operationID string) Annotation {
	return Annotation{OperationID: operationID}
}

// NewDeprecated creates a deprecated annotation
func NewDeprecated() Annotation {
	return Annotation{Deprecated: true}
}

// NewParameter creates a parameter annotation
func NewParameter(name, in string, required bool) ParameterAnnotation {
	return ParameterAnnotation{
		Name:     name,
		In:       in,
		Required: required,
	}
}

// NewResponse creates a response annotation
func NewResponse(description string) ResponseAnnotation {
	return ResponseAnnotation{
		Description: description,
	}
}

// WithDescription adds a description to an annotation
func (a Annotation) WithDescription(description string) Annotation {
	a.Description = description
	return a
}

// WithOperationID adds an operation ID to an annotation
func (a Annotation) WithOperationID(operationID string) Annotation {
	a.OperationID = operationID
	return a
}

// WithTags adds tags to an annotation
func (a Annotation) WithTags(tags ...string) Annotation {
	a.Tags = append(a.Tags, tags...)
	return a
}

// WithSecurity adds security requirements to an annotation
func (a Annotation) WithSecurity(security map[string][]string) Annotation {
	a.Security = append(a.Security, security)
	return a
}

// WithParameter adds a parameter to an annotation
func (a Annotation) WithParameter(param ParameterAnnotation) Annotation {
	a.Parameters = append(a.Parameters, param)
	return a
}

// WithResponse adds a response to an annotation
func (a Annotation) WithResponse(status int, response ResponseAnnotation) Annotation {
	if a.Responses == nil {
		a.Responses = make(map[int]ResponseAnnotation)
	}
	a.Responses[status] = response
	return a
}

// WithConsumes adds content types to an annotation
func (a Annotation) WithConsumes(contentTypes ...string) Annotation {
	a.Consumes = append(a.Consumes, contentTypes...)
	return a
}

// WithProduces adds content types to an annotation
func (a Annotation) WithProduces(contentTypes ...string) Annotation {
	a.Produces = append(a.Produces, contentTypes...)
	return a
}