package api

import (
	"fmt"
	"net/http"

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
	if loader, ok := ctx.(interface{ LoadRequestParam() (contracts.RequestParam, error) }); ok {
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

// HandleError processes API errors and returns appropriate responses
func HandleError(ctx contracts.RequestContext, err ierrors.Error) {
	if err == nil {
		return
	}

	// Handle validation errors using the errors package
	if errors.IsValidationError(err) {
		_, errorMessage, _, fields := errors.HandleValidationError(err)
		ValidationError(ctx, errorMessage, fields)
		return
	}

	// Handle other API errors - try to get error details from response or use defaults
	var errorCode string = "INTERNAL_ERROR"
	var errorMessage string = "Internal server error"
	var httpCode int = http.StatusInternalServerError

	if response := err.Response(); response != nil {
		// Try to extract error details from response
		if resp, ok := response.(map[string]any); ok {
			if code, ok := resp["code"].(string); ok {
				errorCode = code
			}
			if msg, ok := resp["message"].(string); ok {
				errorMessage = msg
			}
		}
	}

	// Use the error's HTTP status code if available
	if err.HttpCode() > 0 {
		httpCode = err.HttpCode()
	}

	ErrorResponse(ctx, errorCode, errorMessage, httpCode)
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