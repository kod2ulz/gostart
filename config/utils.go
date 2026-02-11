package config

import "github.com/google/uuid"

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
		return any(val.UUID).(T)
	default:
		return zero
	}
}
