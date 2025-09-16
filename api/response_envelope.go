package api

import (
	"encoding/json"
	"net/http"
	"reflect"
	"time"

	"github.com/kod2ulz/gostart/contracts"
)

// ResponseEnvelope represents the standardized response format
type ResponseEnvelope struct {
	Success bool        `json:"success"`
	Type    string      `json:"type"`
	Data    any         `json:"data,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
	Time    int64       `json:"time"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// Meta contains pagination and reference metadata
type Meta struct {
	Total      *int64            `json:"total,omitempty"`
	Limit      *int              `json:"limit,omitempty"`
	Offset     *int              `json:"offset,omitempty"`
	References map[string]any    `json:"references,omitempty"`
	Custom     map[string]any    `json:"custom,omitempty"`
}

// ErrorInfo contains error details
type ErrorInfo struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
	Details any               `json:"details,omitempty"`
}

// ResponseHandler is a generic interface for handling different response types
type ResponseHandler interface {
	Handle(ctx contracts.RequestContext) error
}

// SuccessResponse creates a success response envelope
func SuccessResponse(ctx contracts.RequestContext, data any) error {
	if ctx, ok := ctx.(RequestContext); ok {
		envelope := &ResponseEnvelope{
			Success: true,
			Type:    getTypeName(data),
			Data:    data,
			Time:    time.Now().Unix(),
		}

		// Extract metadata from context if available
		if meta := getResponseMetadata(ctx); meta != nil {
			envelope.Meta = meta
		}

		ctx.JSON(http.StatusOK, envelope)
		return nil
	}
	// Fallback for non-RequestContext implementations
	if jsonCtx, ok := ctx.(interface{ JSON(int, any) }); ok {
		jsonCtx.JSON(http.StatusOK, data)
		return nil
	}
	return nil
}

// ListResponse creates a list response envelope
func ListResponse(ctx contracts.RequestContext, data any, total *int64, limit, offset *int) error {
	if ctx, ok := ctx.(RequestContext); ok {
		envelope := &ResponseEnvelope{
			Success: true,
			Type:    getTypeName(data) + "[]",
			Data:    data,
			Time:    time.Now().Unix(),
		}

		// Set pagination metadata
		if total != nil || limit != nil || offset != nil {
			envelope.Meta = &Meta{}
			if total != nil {
				envelope.Meta.Total = total
			}
			if limit != nil {
				envelope.Meta.Limit = limit
			}
			if offset != nil {
				envelope.Meta.Offset = offset
			}
		}

		// Extract additional metadata from context
		if meta := getResponseMetadata(ctx); meta != nil {
			if envelope.Meta == nil {
				envelope.Meta = &Meta{}
			}
			if meta.References != nil {
				envelope.Meta.References = meta.References
			}
			if meta.Custom != nil {
				envelope.Meta.Custom = meta.Custom
			}
		}

		ctx.JSON(http.StatusOK, envelope)
		return nil
	}
	// Fallback for non-RequestContext implementations
	if jsonCtx, ok := ctx.(interface{ JSON(int, any) }); ok {
		jsonCtx.JSON(http.StatusOK, data)
		return nil
	}
	return nil
}

// ErrorResponse creates an error response envelope
func ErrorResponse(ctx contracts.RequestContext, code string, message string, statusCode int) error {
	if ctx, ok := ctx.(RequestContext); ok {
		envelope := &ResponseEnvelope{
			Success: false,
			Type:    "error",
			Time:    time.Now().Unix(),
			Error: &ErrorInfo{
				Code:    code,
				Message: message,
			},
		}
		ctx.JSON(statusCode, envelope)
		return nil
	}
	// Fallback for non-RequestContext implementations
	if jsonCtx, ok := ctx.(interface{ JSON(int, any) }); ok {
		jsonCtx.JSON(statusCode, map[string]string{
			"error": message,
		})
		return nil
	}
	return nil
}

// ValidationError creates a validation error response
func ValidationError(ctx contracts.RequestContext, message string, fields map[string]string) error {
	if ctx, ok := ctx.(RequestContext); ok {
		envelope := &ResponseEnvelope{
			Success: false,
			Type:    "validation_error",
			Time:    time.Now().Unix(),
			Error: &ErrorInfo{
				Code:    "VALIDATION_ERROR",
				Message: message,
				Fields:  fields,
			},
		}
		ctx.JSON(http.StatusBadRequest, envelope)
		return nil
	}
	// Fallback for non-RequestContext implementations
	if jsonCtx, ok := ctx.(interface{ JSON(int, any) }); ok {
		jsonCtx.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":  message,
			"fields": fields,
		})
		return nil
	}
	return nil
}

// SetResponseMetadata sets metadata in the context for response generation
func SetResponseMetadata(ctx contracts.RequestContext, meta *Meta) error {
	if setCtx, ok := ctx.(interface{ SetContextValue(string, any) error }); ok {
		return setCtx.SetContextValue("response_metadata", meta)
	}
	return nil
}

// SetResponseReference sets a reference in the response metadata
func SetResponseReference(ctx contracts.RequestContext, key string, value any) error {
	meta := getResponseMetadata(ctx)
	if meta == nil {
		meta = &Meta{}
	}
	if meta.References == nil {
		meta.References = make(map[string]any)
	}
	meta.References[key] = value
	return SetResponseMetadata(ctx, meta)
}

// SetResponseCustom sets custom metadata
func SetResponseCustom(ctx contracts.RequestContext, key string, value any) error {
	meta := getResponseMetadata(ctx)
	if meta == nil {
		meta = &Meta{}
	}
	if meta.Custom == nil {
		meta.Custom = make(map[string]any)
	}
	meta.Custom[key] = value
	return SetResponseMetadata(ctx, meta)
}

// getResponseMetadata extracts metadata from context
func getResponseMetadata(ctx contracts.RequestContext) *Meta {
	if getCtx, ok := ctx.(interface{ ContextValue(string) any }); ok {
		if meta, ok := getCtx.ContextValue("response_metadata").(*Meta); ok {
			return meta
		}
	}
	return nil
}

// getTypeName returns the type name of the data
func getTypeName(data any) string {
	if data == nil {
		return "void"
	}

	// Use JSON marshaling to get type info
	if jsonData, err := json.Marshal(data); err == nil {
		var typeMap map[string]any
		if json.Unmarshal(jsonData, &typeMap) == nil {
			if typeName, ok := typeMap["__type"].(string); ok {
				return typeName
			}
		}
	}

	// Fallback to type reflection
	t := typeof(data)
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		return typeof(data).Elem().Name() + "[]"
	case reflect.Ptr:
		return typeof(data).Elem().Name()
	default:
		return typeof(data).Name()
	}
}

// Helper function for reflection
func typeof(data any) reflect.Type {
	if data == nil {
		return reflect.TypeOf(nil)
	}
	return reflect.TypeOf(data)
}