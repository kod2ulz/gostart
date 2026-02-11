package contracts

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/markphelps/optional"
)

// TextValue converts pgtype.Text to string
func TextValue(val pgtype.Text, fallback ...string) string {
	if val.Valid {
		return val.String
	} else if len(fallback) > 0 && len(fallback[0]) > 0 {
		return fallback[0]
	}
	return ""
}

// NullableTextValue converts pgtype.Text to *string
func NullableTextValue(val pgtype.Text, fallback ...string) *string {
	if out := TextValue(val, fallback...); len(out) > 0 {
		return &out
	}
	return nil
}

// TimestampValue converts pgtype.Timestamp to time.Time
func TimestampValue(val pgtype.Timestamp, fallback ...time.Time) time.Time {
	if val.Valid {
		return val.Time
	} else if len(fallback) > 0 {
		return fallback[0]
	}
	return time.Time{}
}

// NullableTimestampValue converts pgtype.Timestamp to *time.Time
func NullableTimestampValue(val pgtype.Timestamp, fallback ...time.Time) *time.Time {
	if val.Valid {
		t := val.Time
		return &t
	} else if len(fallback) > 0 && !fallback[0].IsZero() {
		t := fallback[0]
		return &t
	}
	return nil
}

// Int4Value converts pgtype.Int4 to int32
func Int4Value(val pgtype.Int4, fallback ...int32) int32 {
	if val.Valid {
		return val.Int32
	} else if len(fallback) > 0 {
		return fallback[0]
	}
	return 0
}

// NullableInt4Value converts pgtype.Int4 to *int32
func NullableInt4Value(val pgtype.Int4, fallback ...int32) *int32 {
	if val.Valid {
		i := val.Int32
		return &i
	} else if len(fallback) > 0 {
		i := fallback[0]
		return &i
	}
	return nil
}

// Int8Value converts pgtype.Int8 to int64
func Int8Value(val pgtype.Int8, fallback ...int64) int64 {
	if val.Valid {
		return val.Int64
	} else if len(fallback) > 0 {
		return fallback[0]
	}
	return 0
}

// NullableInt8Value converts pgtype.Int8 to *int64
func NullableInt8Value(val pgtype.Int8, fallback ...int64) *int64 {
	if val.Valid {
		i := val.Int64
		return &i
	} else if len(fallback) > 0 {
		i := fallback[0]
		return &i
	}
	return nil
}

// Float4Value converts pgtype.Float4 to float32
func Float4Value(val pgtype.Float4, fallback ...float32) float32 {
	if val.Valid {
		return val.Float32
	} else if len(fallback) > 0 {
		return fallback[0]
	}
	return 0
}

// NullableFloat4Value converts pgtype.Float4 to *float32
func NullableFloat4Value(val pgtype.Float4, fallback ...float32) *float32 {
	if val.Valid {
		f := val.Float32
		return &f
	} else if len(fallback) > 0 {
		f := fallback[0]
		return &f
	}
	return nil
}

// Float8Value converts pgtype.Float8 to float64
func Float8Value(val pgtype.Float8, fallback ...float64) float64 {
	if val.Valid {
		return val.Float64
	} else if len(fallback) > 0 {
		return fallback[0]
	}
	return 0
}

// NullableFloat8Value converts pgtype.Float8 to *float64
func NullableFloat8Value(val pgtype.Float8, fallback ...float64) *float64 {
	if val.Valid {
		f := val.Float64
		return &f
	} else if len(fallback) > 0 {
		f := fallback[0]
		return &f
	}
	return nil
}

// BoolValue converts pgtype.Bool to bool
func BoolValue(val pgtype.Bool, fallback ...bool) bool {
	if val.Valid {
		return val.Bool
	} else if len(fallback) > 0 {
		return fallback[0]
	}
	return false
}

// NullableBoolValue converts pgtype.Bool to *bool
func NullableBoolValue(val pgtype.Bool, fallback ...bool) *bool {
	if val.Valid {
		b := val.Bool
		return &b
	} else if len(fallback) > 0 {
		b := fallback[0]
		return &b
	}
	return nil
}

// UUIDValue converts pgtype.UUID to uuid.UUID
func UUIDValue(val pgtype.UUID, fallback ...uuid.UUID) uuid.UUID {
	if val.Valid {
		return uuid.UUID(val.Bytes)
	} else if len(fallback) > 0 {
		return fallback[0]
	}
	return uuid.Nil
}

// NullableUUIDValue converts pgtype.UUID to *uuid.UUID
func NullableUUIDValue(val pgtype.UUID, fallback ...uuid.UUID) *uuid.UUID {
	if val.Valid {
	 uid := uuid.UUID(val.Bytes)
		return &uid
	} else if len(fallback) > 0 && fallback[0] != uuid.Nil {
	 uid := fallback[0]
		return &uid
	}
	return nil
}

// TextParam converts string to pgtype.Text
func TextParam(val string) pgtype.Text {
	return pgtype.Text{String: val, Valid: true}
}

// OptionalTextParam converts optional.String to pgtype.Text
func OptionalTextParam(val optional.String, fallback ...string) pgtype.Text {
	if !val.Present() {
		if len(fallback) > 0 {
			return pgtype.Text{String: fallback[0], Valid: true}
		}
		return pgtype.Text{}
	} else if str, err := val.Get(); err == nil {
		return pgtype.Text{String: str, Valid: true}
	}
	return pgtype.Text{}
}

// NullableTextParam converts *string to pgtype.Text
func NullableTextParam(val *string, fallback ...string) pgtype.Text {
	if val != nil {
		return pgtype.Text{String: *val, Valid: true}
	} else if len(fallback) > 0 {
		return pgtype.Text{String: fallback[0], Valid: true}
	}
	return pgtype.Text{}
}

// TimestampParam converts time.Time to pgtype.Timestamp
func TimestampParam(val time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{Time: val, Valid: !val.IsZero()}
}

// NullableTimestampParam converts *time.Time to pgtype.Timestamp
func NullableTimestampParam(val *time.Time, fallback ...time.Time) pgtype.Timestamp {
	if val != nil {
		return pgtype.Timestamp{Time: *val, Valid: true}
	} else if len(fallback) > 0 {
		return pgtype.Timestamp{Time: fallback[0], Valid: true}
	}
	return pgtype.Timestamp{}
}

// Int4Param converts int32 to pgtype.Int4
func Int4Param(val int32) pgtype.Int4 {
	return pgtype.Int4{Int32: val, Valid: true}
}

// OptionalInt4Param converts optional.Int to pgtype.Int4
func OptionalInt4Param(val optional.Int, fallback ...int32) pgtype.Int4 {
	if !val.Present() {
		if len(fallback) > 0 {
			return pgtype.Int4{Int32: fallback[0], Valid: true}
		}
		return pgtype.Int4{}
	} else if num, err := val.Get(); err == nil {
		return pgtype.Int4{Int32: int32(num), Valid: true}
	}
	return pgtype.Int4{}
}

// NullableInt4Param converts *int32 to pgtype.Int4
func NullableInt4Param(val *int32, fallback ...int32) pgtype.Int4 {
	if val != nil {
		return pgtype.Int4{Int32: *val, Valid: true}
	} else if len(fallback) > 0 {
		return pgtype.Int4{Int32: fallback[0], Valid: true}
	}
	return pgtype.Int4{}
}

// Int8Param converts int64 to pgtype.Int8
func Int8Param(val int64) pgtype.Int8 {
	return pgtype.Int8{Int64: val, Valid: true}
}

// OptionalInt8Param converts optional.Int64 to pgtype.Int8
func OptionalInt8Param(val optional.Int64, fallback ...int64) pgtype.Int8 {
	if !val.Present() {
		if len(fallback) > 0 {
			return pgtype.Int8{Int64: fallback[0], Valid: true}
		}
		return pgtype.Int8{}
	} else if num, err := val.Get(); err == nil {
		return pgtype.Int8{Int64: num, Valid: true}
	}
	return pgtype.Int8{}
}

// NullableInt8Param converts *int64 to pgtype.Int8
func NullableInt8Param(val *int64, fallback ...int64) pgtype.Int8 {
	if val != nil {
		return pgtype.Int8{Int64: *val, Valid: true}
	} else if len(fallback) > 0 {
		return pgtype.Int8{Int64: fallback[0], Valid: true}
	}
	return pgtype.Int8{}
}

// Float4Param converts float32 to pgtype.Float4
func Float4Param(val float32) pgtype.Float4 {
	return pgtype.Float4{Float32: val, Valid: true}
}

// OptionalFloat4Param converts optional.Float32 to pgtype.Float4
func OptionalFloat4Param(val optional.Float32, fallback ...float32) pgtype.Float4 {
	if !val.Present() {
		if len(fallback) > 0 {
			return pgtype.Float4{Float32: fallback[0], Valid: true}
		}
		return pgtype.Float4{}
	} else if num, err := val.Get(); err == nil {
		return pgtype.Float4{Float32: num, Valid: true}
	}
	return pgtype.Float4{}
}

// NullableFloat4Param converts *float32 to pgtype.Float4
func NullableFloat4Param(val *float32, fallback ...float32) pgtype.Float4 {
	if val != nil {
		return pgtype.Float4{Float32: *val, Valid: true}
	} else if len(fallback) > 0 {
		return pgtype.Float4{Float32: fallback[0], Valid: true}
	}
	return pgtype.Float4{}
}

// Float8Param converts float64 to pgtype.Float8
func Float8Param(val float64) pgtype.Float8 {
	return pgtype.Float8{Float64: val, Valid: true}
}

// OptionalFloat8Param converts optional.Float64 to pgtype.Float8
func OptionalFloat8Param(val optional.Float64, fallback ...float64) pgtype.Float8 {
	if !val.Present() {
		if len(fallback) > 0 {
			return pgtype.Float8{Float64: fallback[0], Valid: true}
		}
		return pgtype.Float8{}
	} else if num, err := val.Get(); err == nil {
		return pgtype.Float8{Float64: num, Valid: true}
	}
	return pgtype.Float8{}
}

// NullableFloat8Param converts *float64 to pgtype.Float8
func NullableFloat8Param(val *float64, fallback ...float64) pgtype.Float8 {
	if val != nil {
		return pgtype.Float8{Float64: *val, Valid: true}
	} else if len(fallback) > 0 {
		return pgtype.Float8{Float64: fallback[0], Valid: true}
	}
	return pgtype.Float8{}
}

// BoolParam converts bool to pgtype.Bool
func BoolParam(val bool) pgtype.Bool {
	return pgtype.Bool{Bool: val, Valid: true}
}

// OptionalBoolParam converts optional.Bool to pgtype.Bool
func OptionalBoolParam(val optional.Bool, fallback ...bool) pgtype.Bool {
	if !val.Present() {
		if len(fallback) > 0 {
			return pgtype.Bool{Bool: fallback[0], Valid: true}
		}
		return pgtype.Bool{}
	} else if b, err := val.Get(); err == nil {
		return pgtype.Bool{Bool: b, Valid: true}
	}
	return pgtype.Bool{}
}

// NullableBoolParam converts *bool to pgtype.Bool
func NullableBoolParam(val *bool, fallback ...bool) pgtype.Bool {
	if val != nil {
		return pgtype.Bool{Bool: *val, Valid: true}
	} else if len(fallback) > 0 {
		return pgtype.Bool{Bool: fallback[0], Valid: true}
	}
	return pgtype.Bool{}
}

// UUIDParam converts uuid.UUID to pgtype.UUID
func UUIDParam(val uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: val, Valid: val != uuid.Nil}
}

// NullableUUIDParam converts *uuid.UUID to pgtype.UUID
func NillableUUIDParam(val *uuid.UUID, fallback ...uuid.UUID) pgtype.UUID {
	if val != nil && *val != uuid.Nil {
		return pgtype.UUID{Bytes: *val, Valid: true}
	} else if len(fallback) > 0 && fallback[0] != uuid.Nil {
		return pgtype.UUID{Bytes: fallback[0], Valid: true}
	}
	return pgtype.UUID{}
}

// NullableUUIDParam converts *uuid.UUID to pgtype.UUID
func NullableUUIDParam(val uuid.NullUUID, fallback ...uuid.UUID) pgtype.UUID {
	if val.Valid && val.UUID != uuid.Nil {
		return pgtype.UUID{Bytes: val.UUID, Valid: true}
	} else if len(fallback) > 0 && fallback[0] != uuid.Nil {
		return pgtype.UUID{Bytes: fallback[0], Valid: true}
	}
	return pgtype.UUID{}
}

// OptionalUUIDParam converts optional.String containing a UUID to pgtype.UUID
func OptionalUUIDParam(val optional.String, fallback ...uuid.UUID) pgtype.UUID {
	if !val.Present() {
		if len(fallback) > 0 && fallback[0] != uuid.Nil {
			return pgtype.UUID{Bytes: fallback[0], Valid: true}
		}
		return pgtype.UUID{}
	}

	if str, err := val.Get(); err == nil {
		if uid, err := uuid.Parse(str); err == nil {
			return pgtype.UUID{Bytes: uid, Valid: true}
		}
	}

	return pgtype.UUID{}
}

// ConvertID converts Value to the specified ID type
func ConvertID[T comparable](val Value) T {
	var zero T
	switch any(zero).(type) {
	case string:
		return any(val.String()).(T)
	case int32:
		return any(int32(val.Int())).(T)
	case int64:
		return any(val.Int64()).(T)
	case uuid.UUID:
		if uid, err := uuid.Parse(val.String()); err == nil {
			return any(uid).(T)
		}
		return zero
	default:
		return zero
	}
}
