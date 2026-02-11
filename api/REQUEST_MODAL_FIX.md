# RequestModal Infinite Recursion Fix

## Problem

When a type embedded `RequestModal[T]` (e.g., `GeoAttributeRequest`), calling `RequestLoad()` would cause infinite recursion and stack overflow:

```go
type GeoAttributeRequest struct {
    ID       uuid.UUID `json:"id"`
    Name     string    `json:"name"`
    api.RequestModal[GeoAttributeRequest]  // This caused infinite recursion
}

// Calling RequestLoad would crash with:
// runtime: goroutine stack exceeds 1000000000-byte limit
```

## Root Cause

The `RequestModal[T].RequestLoad()` method attempted to detect if `T` had a custom `RequestLoad` implementation by creating a `new(T)` instance and type-asserting it. However, when `T` embeds `RequestModal[T]`, Go's method promotion makes the embedded `RequestLoad` method appear as if it belongs to `T`, causing the method to call itself infinitely.

## Solution

**Removed the automatic custom `RequestLoad` detection feature.** The feature was too complex to implement reliably due to Go's method promotion behavior.

### What Changed

**Before:**
```go
func (r RequestModal[T]) RequestLoad(ctx contracts.RequestContext) (param contracts.RequestParam, err error) {
    t := new(T)

    // Check if T has its own RequestLoad method
    if loader, ok := interface{}(t).(interface{ RequestLoad(...) }); ok {
        return loader.RequestLoad(ctx)  // <-- Infinite recursion when T embeds RequestModal[T]
    }

    // Tag-based loading...
}
```

**After:**
```go
func (r RequestModal[T]) RequestLoad(ctx contracts.RequestContext) (param contracts.RequestParam, err error) {
    t := new(T)

    // Tag-driven cascading load order:
    // 1. JSON body (if any field has `json:` tags)
    // 2. Query params (if any field has `query:` tags) - overrides JSON
    // 3. Path params (if any field has `param:` tags) - overrides query
    // 4. Headers (if any field has `header:` tags) - overrides path params
    // 5. Middleware data (if any field has `middleware[name]:` tags) - highest priority
    // ... [tag-based loading logic]
}
```

### Impact

**No Breaking Changes for Most Code**

For types that simply embed `RequestModal[T]`, this fix resolves the infinite recursion bug:

```go
type GeoAttributeRequest struct {
    ID       uuid.UUID `json:"id" validate:"required"`
    Name     string    `json:"name"`
    api.RequestModal[GeoAttributeRequest]
}

// This now works correctly without infinite recursion!
// It uses tag-based loading from RequestModal[T]
```

**Types with Custom RequestLoad (e.g., ListRequestWithID)**

Types that provide custom `RequestLoad` implementations continue to work as before:

```go
type ListRequestWithID[ID ListRequestIdType] struct {
    ID ID
    ListRequest
    RequestModal[ListRequestWithID[ID]]
}

// Custom RequestLoad implementation
func (r ListRequestWithID[ID]) RequestLoad(ctx contracts.RequestContext) (param contracts.RequestParam, err error) {
    var out ListRequestWithID[ID] = ListRequestWithID[ID]{ListRequest: ListRequest{}}

    // Explicitly call ListRequest's RequestLoad (which uses tag-based loading)
    if p, e := out.ListRequest.RequestLoad(ctx); e != nil {
        return param, errors.RequestLoadFailed[ListRequestWithID[ID]](e)
    } else {
        out.ListRequest = p.(ListRequest)
    }

    // Load ID from path parameter
    pathId := string(ctx.Param("id", ""))
    // ... [custom ID loading logic]

    return out, nil
}
```

**Note:** `ListRequestWithID` was already explicitly calling `out.ListRequest.RequestLoad(ctx)`, so it was never using the automatic detection feature. It continues to work as before.

## Benefits

1. ✅ **No More Infinite Recursion**: Types embedding `RequestModal[T]` work correctly
2. ✅ **Simpler Code**: Removed complex and unreliable type detection logic
3. ✅ **Predictable Behavior**: Tag-based loading always works the same way
4. ✅ **No Breaking Changes**: Existing code continues to work

## Testing

Added comprehensive tests in `api/request_modal_test.go`:

- `TestRequestModalNoInfiniteRecursion`: Verifies that embedding `RequestModal[T]` doesn't cause infinite recursion
- `TestRequestModalWithTagLoading`: Verifies tag-based loading works correctly
- `TestRequestModalEmptyRequest`: Verifies empty requests work correctly

## Migration Guide

**If you have a type that embeds `RequestModal[T]`:**

No changes needed! The infinite recursion bug is now fixed.

**If you have a type with custom `RequestLoad` logic:**

Ensure your custom implementation explicitly calls the base `RequestLoad` method:

```go
func (r MyType) RequestLoad(ctx contracts.RequestContext) (param contracts.RequestParam, err error) {
    // Call base RequestLoad for tag-based loading
    baseParam, baseErr := r.RequestModal[MyType].RequestLoad(ctx)
    if baseErr != nil {
        return nil, baseErr
    }

    // Your custom logic here
    // ...
}
```

However, if your type already works (like `ListRequestWithID`), no changes are needed.

## Related Issues

- Fixed infinite recursion in `GeoAttributeRequest`, `GeoRegionRequest`, `GeoTypeRequest`, `GeoUnitRequest`, etc.
- Verified that `ListRequestWithID[ID]` continues to work correctly
