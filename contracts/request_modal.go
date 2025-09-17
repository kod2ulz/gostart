package contracts

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/kod2ulz/gostart/utils"
)

// RequestModal provides a minimal, framework-agnostic default implementation for RequestParam interface.
// It provides basic functionality for request parameter loading and validation using reflection and struct tags.
// This is intended as a base implementation that can be extended by framework-specific implementations.
//
// Use this when you need a basic RequestParam implementation without framework-specific features.
type RequestModal[T RequestParam] struct{}

// Ensure RequestModal implements RequestParam
// Note: This will only work for concrete types that implement RequestParam
// var _ RequestParam = RequestModal[ConcreteRequestParam]{}

// Validate provides default validation using struct tags
func (r RequestModal[T]) Validate(ctx RequestContext) error {
	// Get the value from context and validate it
	if impl, ok := ctx.(*contextImpl); ok {
		if val := impl.Value(r.ContextKey()); val != nil {
			return utils.Validate.Struct(val)
		}
	}
	return nil
}

// RequestLoad provides default request loading using reflection and struct tags
func (r RequestModal[T]) RequestLoad(ctx RequestContext) (RequestParam, error) {
	t := new(T)

	// Use reflection to populate struct fields based on tags
	if err := DefaultRequestLoader(ctx, t); err != nil {
		return nil, err
	}

	// Store in context if possible
	if impl, ok := ctx.(*contextImpl); ok {
		impl.Set(r.ContextKey(), t)
	}
	return *t, nil
}

// ContextKey returns the context key for this parameter type
func (r RequestModal[T]) ContextKey() string {
	// Use type name instead of pointer type to avoid issues
	return fmt.Sprintf("%T", *new(T))
}

// ContextLoad retrieves parameter from standard Go context
func (r RequestModal[T]) ContextLoad(ctx context.Context) (RequestParam, error) {
	val := ctx.Value(r.ContextKey())
	if val == nil {
		return nil, fmt.Errorf("value of type %T with key %s not found in context", new(T), r.ContextKey())
	}
	return val.(RequestParam), nil
}

// DefaultRequestLoader uses reflection to populate struct fields based on tags
func DefaultRequestLoader(ctx RequestContext, target interface{}) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("target must be a non-nil pointer")
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("target must point to a struct")
	}

	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !fieldValue.CanSet() {
			continue
		}

		// Handle different tag types
		if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
			// Load from JSON body
			if err := ctx.ShouldBindJSON(target); err != nil {
				return fmt.Errorf("failed to bind JSON: %w", err)
			}
			return nil // JSON binding handles all fields at once
		}

		if queryTag := field.Tag.Get("query"); queryTag != "" {
			// Load from query parameter
			if value := ctx.Query(queryTag); value.Valid() {
				if err := setFieldValue(fieldValue, value.String()); err != nil {
					return fmt.Errorf("failed to set query field %s: %w", field.Name, err)
				}
			}
		}

		if paramTag := field.Tag.Get("param"); paramTag != "" {
			// Load from path parameter
			if value := ctx.Param(paramTag); value.Valid() {
				if err := setFieldValue(fieldValue, value.String()); err != nil {
					return fmt.Errorf("failed to set param field %s: %w", field.Name, err)
				}
			}
		}

		if headerTag := field.Tag.Get("header"); headerTag != "" {
			// Load from header
			if value := ctx.Header(headerTag); value != "" {
				if err := setFieldValue(fieldValue, value); err != nil {
					return fmt.Errorf("failed to set header field %s: %w", field.Name, err)
				}
			}
		}
	}

	return nil
}

// setFieldValue sets a struct field value from a string using reflection
func setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var intVal int64
		_, err := fmt.Sscanf(value, "%d", &intVal)
		if err != nil {
			return fmt.Errorf("invalid integer value: %s", value)
		}
		field.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var uintVal uint64
		_, err := fmt.Sscanf(value, "%d", &uintVal)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer value: %s", value)
		}
		field.SetUint(uintVal)
	case reflect.Float32, reflect.Float64:
		var floatVal float64
		_, err := fmt.Sscanf(value, "%f", &floatVal)
		if err != nil {
			return fmt.Errorf("invalid float value: %s", value)
		}
		field.SetFloat(floatVal)
	case reflect.Bool:
		boolVal := strings.ToLower(value) == "true" || value == "1"
		field.SetBool(boolVal)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
	return nil
}

// contextImpl is a wrapper to provide Set method for context storage
type contextImpl struct {
	RequestContext
	values map[string]interface{}
}

func (c *contextImpl) Set(key string, value interface{}) {
	if c.values == nil {
		c.values = make(map[string]interface{})
	}
	c.values[key] = value
}

func (c *contextImpl) Value(key string) interface{} {
	return c.values[key]
}
