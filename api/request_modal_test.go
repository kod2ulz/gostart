package api

import (
	"context"
	"testing"

	"github.com/kod2ulz/gostart/contracts"
	"github.com/stretchr/testify/assert"
)

// TestRequestModalNoInfiniteRecursion tests that embedding RequestModal[T] doesn't cause infinite recursion
// This reproduces the bug where GeoAttributeRequest embeds RequestModal[GeoAttributeRequest]
// and RequestLoad would call itself infinitely.
func TestRequestModalNoInfiniteRecursion(t *testing.T) {
	// Define a type that embeds RequestModal[T], similar to GeoAttributeRequest
	type TestRequest struct {
		RequestModal[TestRequest]
		Name string `json:"name" query:"name"`
	}

	req := TestRequest{}

	// Create a mock context
	mockCtx := &mockRequestContext{
		queryParams: map[string]string{"name": "test"},
	}

	// This should NOT cause infinite recursion
	// Before the fix, this would crash with "runtime: goroutine stack exceeds limit"
	result, err := req.RequestLoad(mockCtx)

	// Verify we got a result without infinite recursion
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify the result is the correct type
	typedResult, ok := result.(TestRequest)
	assert.True(t, ok, "Result should be TestRequest type")
	assert.Equal(t, "test", typedResult.Name, "Query parameter should be loaded")
}

// mockRequestContext is a minimal mock for testing RequestLoad
type mockRequestContext struct {
	contracts.RequestContext
	queryParams map[string]string
	pathParams  map[string]string
	headers     map[string]string
	bodyData    []byte
}

func (m *mockRequestContext) Query(key string, defaultValue ...string) contracts.Value {
	if val, ok := m.queryParams[key]; ok {
		return contracts.Value(val)
	}
	if len(defaultValue) > 0 {
		return contracts.Value(defaultValue[0])
	}
	return ""
}

func (m *mockRequestContext) Param(key string, defaultValue ...string) contracts.Value {
	if val, ok := m.pathParams[key]; ok {
		return contracts.Value(val)
	}
	if len(defaultValue) > 0 {
		return contracts.Value(defaultValue[0])
	}
	return ""
}

func (m *mockRequestContext) Header(key string) string {
	return m.headers[key]
}

func (m *mockRequestContext) ShouldBindJSON(obj interface{}) error {
	// For simplicity, not implementing JSON binding in this mock
	return nil
}

func (m *mockRequestContext) Context() context.Context {
	return context.Background()
}

// TestRequestModalWithTagLoading tests that tag-based loading still works correctly
func TestRequestModalWithTagLoading(t *testing.T) {
	type TagTestRequest struct {
		RequestModal[TagTestRequest]
		Name     string `json:"name" query:"name"`
		Age      int    `query:"age"`
		Email    string `header:"email"`
		UserID   string `param:"userId"`
		IsActive bool   `query:"active"`
	}

	req := TagTestRequest{}

	mockCtx := &mockRequestContext{
		queryParams: map[string]string{
			"name":   "John Doe",
			"age":    "30",
			"active": "true",
		},
		pathParams: map[string]string{
			"userId": "user123",
		},
		headers: map[string]string{
			"Email": "john@example.com",
		},
	}

	result, err := req.RequestLoad(mockCtx)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	typedResult, ok := result.(TagTestRequest)
	assert.True(t, ok)

	// Verify all sources were loaded correctly
	assert.Equal(t, "John Doe", typedResult.Name, "Query param 'name' should be loaded")
	assert.Equal(t, 30, typedResult.Age, "Query param 'age' should be parsed as int")
	assert.Equal(t, true, typedResult.IsActive, "Query param 'active' should be parsed as bool")
	assert.Equal(t, "user123", typedResult.UserID, "Path param 'userId' should be loaded")
	assert.Equal(t, "john@example.com", typedResult.Email, "Header 'Email' should be loaded")
}

// TestRequestModalEmptyRequest tests that empty requests work correctly
func TestRequestModalEmptyRequest(t *testing.T) {
	type EmptyRequest struct {
		RequestModal[EmptyRequest]
	}

	req := EmptyRequest{}
	mockCtx := &mockRequestContext{}

	result, err := req.RequestLoad(mockCtx)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	_, ok := result.(EmptyRequest)
	assert.True(t, ok, "Result should be EmptyRequest type")
}
