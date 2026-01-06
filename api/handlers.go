package api

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/errors"
	"github.com/kod2ulz/gostart/ierrors"
)

// This package provides simplified, type-safe handler functions for common API patterns.
// The goal is to reduce boilerplate and provide consistent error handling and response formatting.
//
// Usage examples:
//
//   // Simple single object response
//   router.GET("/users/:id", Handler(func(ctx contracts.RequestContext, param contracts.RequestParam) (User, ierrors.Error) {
//       id := ctx.Param("id")
//       return userService.GetUser(id)
//   }))
//
//   // List response with pagination
//   router.GET("/users", ListHandler(func(ctx contracts.RequestContext, param contracts.RequestParam) ([]User, *int64, ierrors.Error) {
//       return userService.ListUsers(param)
//   }))
//
//   // Simple JSON response without request parameters
//   router.GET("/health", JSONHandler(func(ctx contracts.RequestContext) (Health, ierrors.Error) {
//       return healthService.Check()
//   }))
//

// LoadRequestParam loads request parameters from context
func LoadRequestParam(ctx contracts.RequestContext) (contracts.RequestParam, error) {
	if loader, ok := ctx.(interface {
		LoadRequestParam() (contracts.RequestParam, error)
	}); ok {
		return loader.LoadRequestParam()
	}
	return nil, fmt.Errorf("failed to load request parameters")
}

// extractPagination extracts pagination info from request parameters
func extractPagination(param contracts.RequestParam) (limit, offset *int) {
	if listParam, ok := param.(interface{ GetLimit() int }); ok {
		if l := listParam.GetLimit(); l > 0 {
			limit = &l
		}
	}
	if listParam, ok := param.(interface{ GetOffset() int }); ok {
		if o := listParam.GetOffset(); o > 0 {
			offset = &o
		}
	}
	return
}

// handleRequestError handles common request parameter loading errors
func handleRequestError(ctx contracts.RequestContext, err error) {
	if err != nil {
		ErrorResponse(ctx, "INVALID_REQUEST", err.Error(), http.StatusBadRequest)
	}
}

// RequestHandlerFunc represents a function that handles a request and returns a result
type RequestHandlerFunc[T any] func(ctx contracts.RequestContext, param contracts.RequestParam) (T, ierrors.Error)

// TypedRequestHandlerFunc represents a function with concrete request and response types
// This eliminates the need for type assertions in handlers
type TypedRequestHandlerFunc[P contracts.RequestParam, R any] func(ctx contracts.RequestContext, req P) (R, ierrors.Error)

// TypedListRequestHandlerFunc represents a list handler with concrete request type
type TypedListRequestHandlerFunc[P contracts.RequestParam, R any] func(ctx contracts.RequestContext, req P) ([]R, *int64, ierrors.Error)

// ListRequestHandlerFunc represents a function that handles a request and returns a list with pagination
type ListRequestHandlerFunc[T any] func(ctx contracts.RequestContext, param contracts.RequestParam) ([]T, *int64, ierrors.Error)

// StreamHandlerFunc represents a function that handles a request and returns a streaming response
type StreamHandlerFunc[T any] func(ctx contracts.RequestContext) (<-chan T, ierrors.Error)

// FileHandlerFunc represents a function that handles a request and returns a file path
type FileHandlerFunc func(ctx contracts.RequestContext) (string, ierrors.Error)

// DownloadHandlerFunc represents a function that handles a request and returns a file path and filename
type DownloadHandlerFunc func(ctx contracts.RequestContext) (string, string, ierrors.Error)

// Handler creates a simple handler that returns a single object response
func Handler[T any](handler RequestHandlerFunc[T]) func(contracts.RequestContext) {
	return func(ctx contracts.RequestContext) {
		param, err := LoadRequestParam(ctx)
		if err != nil {
			handleRequestError(ctx, err)
			return
		}

		result, apiErr := handler(ctx, param)
		if apiErr != nil {
			HandleError(ctx, apiErr)
			return
		}

		SuccessResponse(ctx, result)
	}
}

// ListHandler creates a handler that returns a list response with pagination
func ListHandler[T any](handler ListRequestHandlerFunc[T]) func(contracts.RequestContext) {
	return func(ctx contracts.RequestContext) {
		param, err := LoadRequestParam(ctx)
		if err != nil {
			handleRequestError(ctx, err)
			return
		}

		result, total, apiErr := handler(ctx, param)
		if apiErr != nil {
			HandleError(ctx, apiErr)
			return
		}

		limit, offset := extractPagination(param)
		ListResponse(ctx, result, total, limit, offset)
	}
}

// RawHandler creates a handler that doesn't use the response envelope
func RawHandler(handler func(contracts.RequestContext)) func(contracts.RequestContext) {
	return handler
}

// HandleError processes API errors using centralized error handling from errors package
func HandleError(ctx contracts.RequestContext, err ierrors.Error) {
	if err == nil {
		return
	}

	// Capture error location automatically by searching the call stack
	// This finds the actual error origin in user code, not framework internals
	SetErrorLocation(ctx, "", 0)

	// Use centralized error handling from errors package
	errorCode, errorMessage, httpCode, fields := errors.HandleAPIError(err)

	// Handle validation errors specifically to use ValidationError response
	if errors.IsValidationError(err) {
		ValidationError(ctx, errorMessage, fields)
		return
	}

	// Use standard error response for other error types (now with fields support)
	ErrorResponse(ctx, errorCode, errorMessage, httpCode, fields)
}

// StreamHandler creates a handler for streaming responses
func StreamHandler[T any](handler StreamHandlerFunc[T]) func(contracts.RequestContext) {
	return func(ctx contracts.RequestContext) {
		stream, err := handler(ctx)
		if err != nil {
			HandleError(ctx, err)
			return
		}

		if apiCtx, ok := ctx.(RequestContext); ok {
			apiCtx.SetHeader("Content-Type", "text/event-stream")
			apiCtx.SetHeader("Cache-Control", "no-cache")
			apiCtx.SetHeader("Connection", "keep-alive")

			for data := range stream {
				if jsonErr := SuccessResponse(ctx, data); jsonErr != nil {
					break
				}
				// TODO: Add streaming flush functionality
			}
		}
	}
}

// FileHandler creates a handler for file responses
func FileHandler(handler FileHandlerFunc) func(contracts.RequestContext) {
	return func(ctx contracts.RequestContext) {
		filePath, err := handler(ctx)
		if err != nil {
			HandleError(ctx, err)
			return
		}

		if apiCtx, ok := ctx.(RequestContext); ok {
			apiCtx.File(filePath)
		}
	}
}

// DownloadHandler creates a handler for file downloads
func DownloadHandler(handler DownloadHandlerFunc) func(contracts.RequestContext) {
	return func(ctx contracts.RequestContext) {
		filePath, filename, err := handler(ctx)
		if err != nil {
			HandleError(ctx, err)
			return
		}

		if apiCtx, ok := ctx.(RequestContext); ok {
			apiCtx.SetHeader("Content-Disposition", "attachment; filename=\""+filename+"\"")
			apiCtx.File(filePath)
		}
	}
}

// Convenience handlers for common use cases
func JSONHandler[T any](handler func(ctx contracts.RequestContext) (T, ierrors.Error)) func(contracts.RequestContext) {
	return func(ctx contracts.RequestContext) {
		result, err := handler(ctx)
		if err != nil {
			HandleError(ctx, err)
			return
		}
		SuccessResponse(ctx, result)
	}
}

func SimpleHandler(handler func(contracts.RequestContext)) func(contracts.RequestContext) {
	return func(ctx contracts.RequestContext) {
		handler(ctx)
	}
}

// TypedHandler creates a handler with concrete request and response types
// Automatically loads the request parameter and passes it to your handler function
// No type assertions needed in your handler code!
//
// Example:
//   type HelloRequest struct {
//       Name string `json:"name" query:"name"`
//   }
//   type HelloResponse struct {
//       Message string `json:"message"`
//   }
//
//   router.GET("/hello", api.TypedHandler(Hello))
//
//   func Hello(ctx contracts.RequestContext, req HelloRequest) (HelloResponse, ierrors.Error) {
//       // req is already typed as HelloRequest - no type assertion needed!
//       return HelloResponse{Message: "hello " + req.Name}, nil
//   }
func TypedHandler[P contracts.RequestParam, R any](handler TypedRequestHandlerFunc[P, R]) func(contracts.RequestContext) {
	return func(ctx contracts.RequestContext) {
		// Set handler name for logging (get the calling function's name)
		SetHandlerName(ctx, GetCallersHandlerName(1))

		// Load request parameters using RequestModal
		// RequestLoad never fails - it loads what's available from multiple sources
		var modal RequestModal[P]
		loaded, _ := modal.RequestLoad(ctx)

		// Type assert to the concrete type
		param, ok := loaded.(P)
		if !ok {
			HandleError(ctx, errors.RequestLoadFailed[P](fmt.Errorf("failed to cast loaded params to %T", param)))
			return
		}

		// Validate the request parameters before calling the handler
		// This ensures all validation rules are checked before business logic runs
		if validateErr := modal.Validate(ctx); validateErr != nil {
			HandleError(ctx, errors.ValidatorError[P](validateErr))
			return
		}

		// Call the handler with the properly typed request
		result, apiErr := handler(ctx, param)
		if apiErr != nil {
			HandleError(ctx, apiErr)
			return
		}

		SuccessResponse(ctx, result)
	}
}

// TypedHandlerWithTypes is like TypedHandler but also returns type information for documentation
// Use this when you want automatic OpenAPI schema generation
//
// Example:
//   router.GET("/hello", api.TypedHandlerWithTypes(Hello), api.WithAnnotation(...))
func TypedHandlerWithTypes[P contracts.RequestParam, R any](handler TypedRequestHandlerFunc[P, R]) (HandlerFunc, RouteOption) {
	// Capture type information for documentation
	var p P
	var r R

	typeOption := WithTypes(
		reflect.TypeOf(p),
		reflect.TypeOf(r),
		false, // not a list
	)

	return TypedHandler[P, R](handler), typeOption
}

// TypedListHandler creates a list handler with concrete request and response types
// Automatically loads request parameters and handles pagination
//
// Example:
//   type ListUsersRequest struct {
//       api.ListRequest
//       Search string `json:"search" query:"search"`
//   }
//
//   router.GET("/users", api.TypedListHandler(ListUsers))
//
//   func ListUsers(ctx contracts.RequestContext, req ListUsersRequest) ([]User, *int64, ierrors.Error) {
//       // req is already typed as ListUsersRequest
//       // req.Limit, req.Offset are available from embedded ListRequest
//       return userService.SearchUsers(req.Search, req.Limit, req.Offset)
//   }
func TypedListHandler[P contracts.RequestParam, R any](handler TypedListRequestHandlerFunc[P, R]) func(contracts.RequestContext) {
	return func(ctx contracts.RequestContext) {
		// Set handler name for logging
		SetHandlerName(ctx, GetCallersHandlerName(1))

		// Create a zero-value instance of the request type
		var param P

		// Load request parameters using RequestModal
		var modal RequestModal[P]
		loaded, err := modal.RequestLoad(ctx)
		if err != nil {
			HandleError(ctx, errors.RequestLoadFailed[P](err))
			return
		}

		// Type assert to the concrete type
		param, ok := loaded.(P)
		if !ok {
			HandleError(ctx, errors.RequestLoadFailed[P](fmt.Errorf("failed to cast loaded params to %T", param)))
			return
		}

		// Call the handler with the properly typed request
		result, total, apiErr := handler(ctx, param)
		if apiErr != nil {
			HandleError(ctx, apiErr)
			return
		}

		limit, offset := extractPagination(param)
		ListResponse(ctx, result, total, limit, offset)
	}
}

// TypedListHandlerWithTypes is like TypedListHandler but also returns type information for documentation
// Use this when you want automatic OpenAPI schema generation
//
// Example:
//   handler, typeOpt := api.TypedListHandlerWithTypes(ListUsers)
//   router.GET("/users", handler, api.WithAnnotation(...), typeOpt)
func TypedListHandlerWithTypes[P contracts.RequestParam, R any](handler TypedListRequestHandlerFunc[P, R]) (HandlerFunc, RouteOption) {
	// Capture type information for documentation
	var p P
	var r R

	typeOption := WithTypes(
		reflect.TypeOf(p),
		reflect.TypeOf(r),
		true, // is a list
	)

	return TypedListHandler[P, R](handler), typeOption
}
