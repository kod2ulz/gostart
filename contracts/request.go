package contracts

import (
	"context"

	"github.com/kod2ulz/gostart/ierrors"
)

// RequestParam defines the interface for request parameter structs.
// Implementations should embed contracts.RequestModal[T] for default behavior.
type RequestParam interface {
	// Validate validates the request parameters using struct tags
	Validate(ctx RequestContext) error

	// RequestLoad loads and populates the request from the HTTP context
	RequestLoad(ctx RequestContext) (RequestParam, error)

	// ContextKey returns the key used to store this parameter in context
	ContextKey() string

	// MetadataContextKey returns the key used to store response metadata
	MetadataContextKey() string

	// ReferencesContextKey returns the key used to store response references
	ReferencesContextKey() string

	// SetResponseMetadata stores metadata in the context for response generation
	SetResponseMetadata(ctx RequestContext, meta *Metadata) error

	// SetResponseReference stores a reference in the context for response generation
	SetResponseReference(ctx RequestContext, key string, value any) error

	// ContextLoad retrieves the parameter from a standard Go context
	ContextLoad(ctx context.Context) (RequestParam, error)
}

// RequestContext defines the interface for accessing request data in a framework-agnostic way.
// This interface abstracts away the underlying web framework's context.
type RequestContext interface {
	// Query returns the value of a URL query parameter with optional default
	Query(key string, defaultValue ...string) Value

	// Param returns the value of a URL path parameter with optional default
	Param(key string, defaultValue ...string) Value

	// Header returns the value of an HTTP header
	Header(key string) string

	// ShouldBindJSON binds the request body to the given object
	ShouldBindJSON(obj interface{}) error

	// Context returns the underlying standard Go context.Context
	Context() context.Context
}

// Value represents a parameter value that can be converted to various types
type Value string

func (v Value) Valid() bool {
	return v != ""
}

func (v Value) String() string {
	return string(v)
}

func (v Value) Int() int {
	// Implementation needed - for now return 0
	return 0
}

func (v Value) Int64() int64 {
	// Implementation needed - for now return 0
	return 0
}

func (v Value) Float64() float64 {
	// Implementation needed - for now return 0
	return 0
}

func (v Value) Bool() bool {
	// Implementation needed - for now return false
	return false
}

// ParamsFromContext retrieves a RequestParam from a standard Go context
func ParamsFromContext[P RequestParam](ctx context.Context) (P, ierrors.Error) {
	// Implementation needed - will be filled later
	var zero P
	return zero, nil
}