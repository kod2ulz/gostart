# Validation Error Improvements

## Problems Identified

1. **Only first validation error shown** - Users had to submit requests multiple times, fixing one field at a time
2. **Confusing field names in errors** - Error messages used Go struct field names (e.g., "UnitID") instead of JSON tag names (e.g., "unitId")
3. **No validation in TypedHandler** - Requests with missing required fields were passed to service handlers, causing database errors

## Solutions Implemented

### 1. JSON Tag Names in Validation Errors (`utils/validator.go`)

**Problem:**
```go
type GeoAttributeRequest struct {
    UnitID   uuid.UUID `json:"unitId" validate:"required"`
}
// Error message said: "UnitID is required"
// User tried: { "UnitID": "..." } ❌ Doesn't work!
// User should use: { "unitId": "..." } ✅
```

**Solution:**
```go
func init() {
    Validate = validator.New()

    // Register custom tag name function to use JSON tags
    Validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
        name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
        if name == "-" {
            return ""
        }
        return name
    })
}
```

**Result:**
```json
{
  "error": {
    "message": "Validation failed for 4 field(s)",
    "fields": {
      "id": "id is a required field",
      "unitId": "unitId is a required field",
      "category": "category is a required field",
      "type": "type is a required field"
    }
  }
}
```

### 2. Return All Validation Errors (`errors/errors.go`)

**Problem:**
```go
// Before: Only first error returned
{
  "error": {
    "message": "ID is required"
  }
}
// User fixes ID, resubmits
{
  "error": {
    "message": "UnitID is required"
  }
}
// ... frustrating cycle continues
```

**Solution:**
```go
func parseValidationErrors[T any](validationErrors validator.ValidationErrors) ierrors.Error {
    fields := make(map[string]string)
    details := make(map[string]any)
    errorMessages := make([]string, 0, len(validationErrors))

    // Iterate through ALL validation errors
    for _, err := range validationErrors {
        fieldName := err.Field()      // JSON tag name
        rule := err.Tag()             // Validation rule
        param := err.Param()          // Rule parameter

        // Generate user-friendly error message
        errorMsg := getValidationMessage(fieldName, rule, param)
        fields[fieldName] = errorMsg
        errorMessages = append(errorMessages, errorMsg)

        // Add detailed information
        details[fieldName] = map[string]any{
            "rule":    rule,
            "message": errorMsg,
            "param":   param,
        }
    }

    // Create summary message
    var summaryMessage string
    if len(errorMessages) == 1 {
        summaryMessage = errorMessages[0]
    } else {
        summaryMessage = fmt.Sprintf("Validation failed for %d field(s)", len(fields))
    }

    // Return error with ALL validation errors
    // ...
}
```

**Result:**
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validation failed for 4 field(s)",
    "params": {
      "id": {"rule": "required", "message": "id is a required field"},
      "unitId": {"rule": "required", "message": "unitId is a required field"},
      "category": {"rule": "required", "message": "category is a required field"},
      "type": {"rule": "required", "message": "type is a required field"}
    }
  }
}
```

### 3. Validation in TypedHandler (`api/handlers.go`)

**Problem:**
```go
// Before: No validation in TypedHandler
func TypedHandler[P, R](handler TypedRequestHandlerFunc[P, R]) func(contracts.RequestContext) {
    return func(ctx contracts.RequestContext) {
        modal := RequestModal[P]{}
        loaded, _ := modal.RequestLoad(ctx)
        param, _ := loaded.(P)

        // ❌ No validation! Handler called even with invalid data
        result, apiErr := handler(ctx, param)
        // ...
    }
}

// Service handler had to manually validate (or not at all!)
func (s *Service) Create(ctx RequestContext, param Request) (Response, ierrors.Error) {
    // ❌ param might have invalid/missing data
    db.Create(param.CreateParams())  // 💥 DB error: null value in column violates not-null constraint
}
```

**Solution:**
```go
// After: Automatic validation in TypedHandler
func TypedHandler[P, R](handler TypedRequestHandlerFunc[P, R]) func(contracts.RequestContext) {
    return func(ctx contracts.RequestContext) {
        modal := RequestModal[P]{}
        loaded, _ := modal.RequestLoad(ctx)
        param, _ := loaded.(P)

        // ✅ Validate before calling handler
        if validateErr := modal.Validate(ctx); validateErr != nil {
            HandleError(ctx, errors.ValidatorError[P](validateErr))
            return  // Stop here, don't call handler
        }

        // ✅ Only called if validation passes
        result, apiErr := handler(ctx, param)
        // ...
    }
}
```

**Result:**
- Validation happens **before** service layer
- Invalid requests are rejected **before** database calls
- Service handlers can assume valid input

## Before vs After

### Before (Poor UX)
```bash
# Request 1: POST /api/admin/geo/attributes -d ''
{
  "error": {
    "message": "ID is required"
  }
}

# Request 2: Fix ID, POST with { "id": "..." }
{
  "error": {
    "message": "UnitID is required"  # ❌ Still uses Go field name!
  }
}

# Request 3: Fix UnitID, POST with { "id": "...", "UnitID": "..." }
# ❌ Wrong field name! JSON didn't load, but validation passed somehow
# 💥 Database error: null value in column "unit_id" violates not-null constraint
```

### After (Great UX)
```bash
# Request 1: POST /api/admin/geo/attributes -d ''
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validation failed for 4 field(s)",
    "params": {
      "id": {"rule": "required", "message": "id is a required field"},
      "unitId": {"rule": "required", "message": "unitId is a required field"},
      "category": {"rule": "required", "message": "category is a required field"},
      "type": {"rule": "required", "message": "type is a required field"}
    }
  }
}

# ✅ User sees ALL errors at once
# ✅ Field names match JSON tags (unitId not UnitID)
# ✅ Clear validation rules for each field
# ✅ No database errors for invalid data
```

## Files Changed

1. **`utils/validator.go`** - Added `RegisterTagNameFunc` to use JSON tag names
2. **`errors/errors.go`** - Added `parseValidationErrors` to return all validation errors
3. **`api/handlers.go`** - Added validation call in `TypedHandler` before invoking handler
4. **`errors/validation_errors_test.go`** - Tests to verify all improvements

## Benefits

1. ✅ **Better Developer Experience** - See all validation errors at once
2. ✅ **Less Confusion** - Error messages use JSON field names
3. ✅ **Prevents Database Errors** - Invalid data rejected before DB calls
4. ✅ **Consistent Validation** - Automatic for all TypedHandler routes
5. ✅ **Backward Compatible** - Existing code continues to work

## Migration Notes

**No code changes required!** The improvements are automatic:

1. Existing validation tags continue to work
2. `TypedHandler` now validates automatically (no manual `Validate()` calls needed in service handlers)
3. Error responses include all validation errors with proper field names

**Optional: Remove manual validation** from service handlers since it's now done in `TypedHandler`:

```go
// Before
func (s *Service) Create(ctx RequestContext, param Request) (Response, ierrors.Error) {
    if err := utils.Validate.Struct(param); err != nil {
        return nil, errors.ValidatorError[Request](err)  // No longer needed!
    }
    // Business logic...
}

// After
func (s *Service) Create(ctx RequestContext, param Request) (Response, ierrors.Error) {
    // Validation already done in TypedHandler ✅
    // Just do business logic...
}
```

## Testing

All tests pass:
```bash
go test ./errors -run TestErrors -v
# PASS: All 3 validation error tests pass
```

Tests verify:
- ✅ Multiple validation errors returned at once
- ✅ Field names use JSON tags (camelCase) not Go struct names (PascalCase)
- ✅ Clear error messages for single validation failures
