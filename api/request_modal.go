package api

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/kod2ulz/gostart/config"
	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/errors"
	"github.com/kod2ulz/gostart/utils"
)

// RequestModal provides a comprehensive, framework-enhanced implementation for RequestParam interface.
// It extends contracts.RequestModal with framework-specific features and response handling capabilities.
// This is the recommended implementation for most use cases as it provides both request processing
// and response metadata handling in a single, convenient package.
//
// Features:
// - Framework-specific request loading (JSON body, query params, path params, headers)
// - Enhanced validation with context-aware error handling
// - Response metadata and reference management
// - Gin-specific optimizations and utilities
//
// Use this when building API handlers that need full request/response lifecycle support.
type _t struct{ RequestModal[_t] }

var _ contracts.RequestParam = RequestModal[_t]{}

type RequestModal[T contracts.RequestParam] struct {
	contracts.RequestModal[T]
}

func (r RequestModal[T]) Validate(ctx contracts.RequestContext) error {
	if ctxSetter, ok := ctx.(interface{ Value(interface{}) interface{} }); ok {
		return utils.Validate.Struct(ctxSetter.Value(r.ContextKey()))
	}
	return fmt.Errorf("cannot validate: invalid context type")
}

func (r RequestModal[T]) RequestLoad(ctx contracts.RequestContext) (param contracts.RequestParam, err error) {
	t := new(T)

	// Tag-driven cascading load order:
	// 1. JSON body (if any field has `json:` tags)
	// 2. Query params (if any field has `query:` tags) - overrides JSON
	// 3. Path params (if any field has `param:` tags) - overrides query
	// 4. Headers (if any field has `header:` tags) - overrides path params
	// 5. Middleware data (if any field has `middleware[name]:` tags) - highest priority

	// Step 1: Check if we should load JSON body
	if r.hasTag("json") {
		_ = r.LoadFromJsonBody(ctx, t) // Ignore errors - might be GET with no body
	}

	// Step 2: Load from query params (overrides JSON)
	if r.hasTag("query") {
		r.loadFromQueryParams(ctx, t)
	}

	// Step 3: Load from path params (overrides query)
	if r.hasTag("param") {
		r.loadFromPathParams(ctx, t)
	}

	// Step 4: Load from headers (overrides path params)
	if r.hasTag("header") {
		r.loadFromHeaders(ctx, t)
	}

	// Step 5: Load from middleware data (highest priority - overrides everything)
	if r.hasMiddlewareTag() {
		r.loadFromMiddleware(ctx, t)
	}

	// Store in context for later access
	if ctxSetter, ok := ctx.(interface{ Set(string, interface{}) }); ok {
		ctxSetter.Set((*t).ContextKey(), t)
	}

	return *t, nil
}

func (r RequestModal[T]) LoadFromJsonBody(ctx contracts.RequestContext, out interface{}) (err error) {
	if err = ctx.ShouldBindJSON(out); err != nil {
		return errors.Wrapf(err, "failed to load json body into %T from request", out)
	}
	return
}

func (r RequestModal[T]) ContextKey() string {
	var t = new(T)
	return fmt.Sprintf("%T", t)
}

func (p RequestModal[T]) ContextLoad(ctx context.Context) (out contracts.RequestParam, err error) {
	val := ctx.Value(p.ContextKey())
	if val == nil {
		return out, errors.Errorf("value of %T with key %s was %v in context", p, p.ContextKey(), val)
	}
	return val.(contracts.RequestParam), nil
}

func (p RequestModal[T]) LoadFromContext(ctx context.Context, out contracts.RequestParam) (err error) {
	var param contracts.RequestParam
	if out == nil {
		return fmt.Errorf("out is nil")
	} else if param, err = (*new(T)).ContextLoad(ctx); err != nil {
		return errors.Wrapf(err, "Failed to load %T from context", out)
	} else if param == nil {
		if param = ctx.Value(p.ContextKey()).(contracts.RequestParam); param == nil {
			return fmt.Errorf("Got %v when loading %T from context", out, out)
		}
	}
	utils.StructCopy(param, out)
	return
}

func (p RequestModal[T]) FromContext(ctx context.Context, out *T) (err error) {
	if out == nil {
		return fmt.Errorf("out is nil")
	} else if val := ctx.Value(p.ContextKey()); val == nil {
		return fmt.Errorf("value of %T with key %s was %v in context", *out, p.ContextKey(), val)
	} else if param, ok := val.(T); ok {
		*out = param
	} else {
		return fmt.Errorf("failed to cast %T to %T ", val, *out)
	}
	return
}

func (p RequestModal[T]) InContext(ctx context.Context, in T) context.Context {
	return context.WithValue(ctx, in.ContextKey(), in)
}

func (p RequestModal[T]) Query(ctx contracts.RequestContext, name string, _default ...string) (out config.Value) {
	if v := ctx.Query(name); v.Valid() {
		return config.Value(v.String())
	} else if len(_default) > 0 {
		return config.Value(_default[0])
	}
	return ""
}

func (p RequestModal[T]) Path(ctx contracts.RequestContext, name string, _default ...string) (out config.Value) {
	if v := ctx.Param(name); v.Valid() {
		return config.Value(v.String())
	} else if len(_default) > 0 {
		return config.Value(_default[0])
	}
	return ""
}

func (p RequestModal[T]) Debug(o any) {
	fmt.Printf("%T.debug(): %+v\n", p, o)
}

func (p RequestModal[T]) Headers(ctx contracts.RequestContext, names ...string) (out map[string]string) {
	out = make(map[string]string)
	if len(names) == 0 {
		return
	}
	for _, header := range names {
		if header := strings.Trim(header, " "); header == "" {
			continue
		} else if val := ctx.Header(header); val != "" {
			out[header] = val
		}
	}
	return
}

func (p RequestModal[T]) Authorization(ctx contracts.RequestContext) (out string) {
	return ctx.Header("Authorization")
}

func (p RequestModal[T]) WithHeaderValues(ctx contracts.RequestContext, headers ...string) context.Context {
	if len(headers) == 0 {
		return ctx.Context()
	}
	headerValues := p.Headers(ctx, headers...)
	if len(headerValues) == 0 {
		return ctx.Context()
	}
	stdCtx := ctx.Context()
	for k, v := range headerValues {
		stdCtx = context.WithValue(stdCtx, k, v)
	}
	return stdCtx
}

// hasTag checks if any field in the struct has the given tag key
func (r RequestModal[T]) hasTag(tagKey string) bool {
	t := new(T)
	typ := reflect.TypeOf(t).Elem()
	if typ.Kind() == reflect.Struct {
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			// Check both direct field and embedded fields
			if field.Anonymous {
				// Recursively check embedded fields
				embeddedType := field.Type
				if embeddedType.Kind() == reflect.Ptr {
					embeddedType = embeddedType.Elem()
				}
				if embeddedType.Kind() == reflect.Struct {
					for j := 0; j < embeddedType.NumField(); j++ {
						if tag := embeddedType.Field(j).Tag.Get(tagKey); tag != "" && tag != "-" {
							return true
						}
					}
				}
			} else {
				// Check direct field
				if tag := field.Tag.Get(tagKey); tag != "" && tag != "-" {
					return true
				}
			}
		}
	}
	return false
}

// loadFromQueryParams loads struct fields from query parameters
func (r RequestModal[T]) loadFromQueryParams(ctx contracts.RequestContext, out interface{}) {
	val := reflect.ValueOf(out).Elem()
	typ := val.Type()

	if typ.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		if !fieldVal.CanSet() {
			continue
		}

		tag := field.Tag.Get("query")
		if tag == "" || tag == "-" {
			continue
		}

		// Get query parameter value
		queryValue := ctx.Query(tag, "").String()
		if queryValue == "" {
			continue
		}

		// Set the field value based on type
		r.setFieldValue(fieldVal, field.Type, queryValue)
	}
}

// loadFromPathParams loads struct fields from URL path parameters
func (r RequestModal[T]) loadFromPathParams(ctx contracts.RequestContext, out interface{}) {
	val := reflect.ValueOf(out).Elem()
	typ := val.Type()

	if typ.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		if !fieldVal.CanSet() {
			continue
		}

		tag := field.Tag.Get("param")
		if tag == "" || tag == "-" {
			continue
		}

		// Get path parameter value
		paramValue := ctx.Param(tag, "").String()
		if paramValue == "" {
			continue
		}

		// Set the field value based on type
		r.setFieldValue(fieldVal, field.Type, paramValue)
	}
}

// loadFromHeaders loads struct fields from HTTP headers
func (r RequestModal[T]) loadFromHeaders(ctx contracts.RequestContext, out interface{}) {
	val := reflect.ValueOf(out).Elem()
	typ := val.Type()

	if typ.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		if !fieldVal.CanSet() {
			continue
		}

		tag := field.Tag.Get("header")
		if tag == "" || tag == "-" {
			continue
		}

		// Get header value (case-insensitive, convert to Title-Case for HTTP)
		headerName := toHeaderName(tag)
		headerValue := ctx.Header(headerName)
		if headerValue == "" {
			continue
		}

		// Set the field value based on type
		r.setFieldValue(fieldVal, field.Type, headerValue)
	}
}

// setFieldValue sets a struct field value from a string based on the field type
func (r RequestModal[T]) setFieldValue(fieldVal reflect.Value, fieldType reflect.Type, value string) {
	switch fieldType.Kind() {
	case reflect.String:
		fieldVal.SetString(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			fieldVal.SetInt(intVal)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if uintVal, err := strconv.ParseUint(value, 10, 64); err == nil {
			fieldVal.SetUint(uintVal)
		}

	case reflect.Float32, reflect.Float64:
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			fieldVal.SetFloat(floatVal)
		}

	case reflect.Bool:
		if boolVal, err := strconv.ParseBool(value); err == nil {
			fieldVal.SetBool(boolVal)
		}
	}
}

// toHeaderName converts a snake_case or camelCase string to HTTP Header format (Title-Case)
func toHeaderName(s string) string {
	// Convert to Title-Case with hyphens
	parts := strings.Split(strings.TrimSpace(s), "_")
	if len(parts) == 1 {
		// Try camelCase or kebab-case
		parts = strings.Split(s, "-")
	}

	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, "-")
}

// hasMiddlewareTag checks if any field has middleware[name]:key format tags
func (r RequestModal[T]) hasMiddlewareTag() bool {
	typ := reflect.TypeFor[T]()

	if typ.Kind() == reflect.Struct {
		return r.hasTagWithPrefix("middleware")
	}
	return false
}

// loadFromMiddleware loads struct fields from middleware data stored in context
// Tag format: middleware[middlewareName]:"fieldName" or middleware[middlewareName]:fieldName
func (r RequestModal[T]) loadFromMiddleware(ctx contracts.RequestContext, out interface{}) {
	val := reflect.ValueOf(out).Elem()
	typ := val.Type()

	if typ.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		if !fieldVal.CanSet() {
			continue
		}

		// Check for middleware tag
		tag := field.Tag.Get("middleware")
		if tag == "" || tag == "-" {
			continue
		}

		// Parse middleware[name]:key format
		middlewareName, key, err := parseMiddlewareTag(tag)
		if err != nil {
			continue
		}

		// Get middleware data from context
		middlewareData := r.getMiddlewareData(ctx, middlewareName)
		if middlewareData == nil {
			continue
		}

		// Extract the specified field from middleware data
		value := r.extractFieldValue(middlewareData, key)
		if value == nil {
			continue
		}

		// Set the field value
		r.setFieldValueFromInterface(fieldVal, field.Type, value)
	}
}

// parseMiddlewareTag parses middleware[name]:key or middleware[name]:"key" format
// Returns: middlewareName, key, error
func parseMiddlewareTag(tag string) (string, string, error) {
	// Format: middleware[name]:key or middleware[name]:"key"
	if !strings.HasPrefix(tag, "middleware[") {
		return "", "", fmt.Errorf("invalid middleware tag format: %s", tag)
	}

	// Extract the part between [ and ]
	start := strings.Index(tag, "[")
	end := strings.Index(tag, "]")
	if start == -1 || end == -1 || end <= start {
		return "", "", fmt.Errorf("invalid middleware tag format: %s", tag)
	}

	middlewareName := tag[start+1 : end]

	// Extract the key after ] and optional :
	keyPart := tag[end+1:]
	if after, ok :=strings.CutPrefix(keyPart, ":"); ok  {
		key := after
		// Remove quotes if present
		key = strings.Trim(key, `"`)
		return middlewareName, key, nil
	}

	// If no explicit key, use the key as-is
	if keyPart != "" {
		return middlewareName, keyPart, nil
	}

	return "", "", fmt.Errorf("no key specified in middleware tag: %s", tag)
}

// getMiddlewareData retrieves middleware data from context
// Middleware should store data using key format: "middleware:name"
func (r RequestModal[T]) getMiddlewareData(ctx contracts.RequestContext, middlewareName string) interface{} {
	// Try to get from context using standard key format
	contextKey := "middleware:" + middlewareName

	if valueGetter, ok := ctx.(interface{ Value(interface{}) interface{} }); ok {
		data := valueGetter.Value(contextKey)
		if data != nil {
			return data
		}
	}

	return nil
}

// extractFieldValue extracts a field value from a struct/map using dot notation
// Supports: "fieldName", "nested.field", "mapKey"
func (r RequestModal[T]) extractFieldValue(data interface{}, key string) interface{} {
	if data == nil || key == "" {
		return nil
	}

	// Handle dot notation for nested fields
	parts := strings.Split(key, ".")
	current := reflect.ValueOf(data)

	for i, part := range parts {
		// Check if current is a pointer, dereference it
		if current.Kind() == reflect.Ptr {
			if current.IsNil() {
				return nil
			}
			current = current.Elem()
		}

		switch current.Kind() {
		case reflect.Struct:
			field := current.FieldByName(part)
			if !field.IsValid() {
				// Try lowercase/camelCase version
				field = current.FieldByName(toCamelCase(part))
			}
			if !field.IsValid() {
				return nil
			}

			// If this is the last part, return the value
			if i == len(parts)-1 {
				return field.Interface()
			}

			// Otherwise, continue navigating
			current = field

		case reflect.Map:
			mapKeys := current.MapKeys()
			for _, mapKey := range mapKeys {
				if fmt.Sprintf("%v", mapKey.Interface()) == part {
					value := current.MapIndex(mapKey)
					if i == len(parts)-1 {
						return value.Interface()
					}
					current = value
					break
				}
			}
			return nil

		case reflect.Interface:
			// Try to convert to concrete type
			current = reflect.ValueOf(current.Interface())
			if current.Kind() == reflect.Ptr && !current.IsNil() {
				current = current.Elem()
			}
			// Retry with this part
			i-- // Retry same part with new current

		default:
			return nil
		}
	}

	return current.Interface()
}

// setFieldValueFromInterface sets a field value from an interface{}
func (r RequestModal[T]) setFieldValueFromInterface(fieldVal reflect.Value, fieldType reflect.Type, value interface{}) {
	if value == nil {
		return
	}

	val := reflect.ValueOf(value)

	// If types match directly, set it
	if val.Type().AssignableTo(fieldType) {
		fieldVal.Set(val)
		return
	}

	// Try to convert based on field type
	switch fieldType.Kind() {
	case reflect.String:
		if strVal, ok := value.(string); ok {
			fieldVal.SetString(strVal)
		} else {
			fieldVal.SetString(fmt.Sprintf("%v", value))
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := value.(type) {
		case int64:
			fieldVal.SetInt(v)
		case int:
			fieldVal.SetInt(int64(v))
		case float64:
			fieldVal.SetInt(int64(v))
		case string:
			if intVal, err := strconv.ParseInt(v, 10, 64); err == nil {
				fieldVal.SetInt(intVal)
			}
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch v := value.(type) {
		case uint64:
			fieldVal.SetUint(v)
		case uint:
			fieldVal.SetUint(uint64(v))
		case int64:
			fieldVal.SetUint(uint64(v))
		case string:
			if uintVal, err := strconv.ParseUint(v, 10, 64); err == nil {
				fieldVal.SetUint(uintVal)
			}
		}

	case reflect.Float32, reflect.Float64:
		switch v := value.(type) {
		case float64:
			fieldVal.SetFloat(v)
		case float32:
			fieldVal.SetFloat(float64(v))
		case int64:
			fieldVal.SetFloat(float64(v))
		case string:
			if floatVal, err := strconv.ParseFloat(v, 64); err == nil {
				fieldVal.SetFloat(floatVal)
			}
		}

	case reflect.Bool:
		if boolVal, ok := value.(bool); ok {
			fieldVal.SetBool(boolVal)
		}
	}
}

// hasTagWithPrefix checks if any field has a tag with the given prefix
// e.g., prefix "middleware" checks for tags like "middleware[name]:key"
func (r RequestModal[T]) hasTagWithPrefix(prefix string) bool {
	typ := reflect.TypeFor[T]()

	if typ.Kind() != reflect.Struct {
		return false
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Check direct field
		tag := field.Tag.Get(prefix)
		if tag != "" && tag != "-" {
			return true
		}

		// Check embedded fields
		if field.Anonymous {
			embeddedType := field.Type
			if embeddedType.Kind() == reflect.Ptr {
				embeddedType = embeddedType.Elem()
			}
			if embeddedType.Kind() == reflect.Struct {
				for j := 0; j < embeddedType.NumField(); j++ {
					tag := embeddedType.Field(j).Tag.Get(prefix)
					if tag != "" && tag != "-" {
						return true
					}
				}
			}
		}
	}

	return false
}

// toCamelCase converts snake_case or kebab-case to CamelCase
func toCamelCase(s string) string {
	parts := strings.Split(strings.TrimSpace(s), "_")
	if len(parts) == 1 {
		parts = strings.Split(s, "-")
	}

	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, "")
}
