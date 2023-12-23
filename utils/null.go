package utils

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/markphelps/optional"
)

var Null nullUtils

type nullUtils struct{}

func (nullUtils) String(val string) sql.NullString {
	return sql.NullString{Valid: true, String: val}
}

func (nullUtils) OptionalString(str optional.String) (out sql.NullString) {
	if val, err := str.Get(); err == nil {
		return sql.NullString{Valid: true, String: val}
	}
	return 
}

func (nullUtils) UUID(val uuid.UUID) uuid.NullUUID {
	return uuid.NullUUID{Valid: true, UUID: val}
}

func (nullUtils) Int32(val int32) sql.NullInt32 {
	return sql.NullInt32{Valid: true, Int32: val}
}

func (nullUtils) Int64(val int64) sql.NullInt64 {
	return sql.NullInt64{Valid: true, Int64: val}
}

func (nullUtils) Float64(val float64) sql.NullFloat64 {
	return sql.NullFloat64{Valid: true, Float64: val}
}

func (nullUtils) Time(val time.Time) sql.NullTime {
	return sql.NullTime{Valid: true, Time: val}
}

func (nullUtils) Bool(val bool) sql.NullBool {
	return sql.NullBool{Valid: true, Bool: val}
}

func (nullUtils) OptionalBool(str optional.Bool) (out sql.NullBool) {
	if val, err := str.Get(); err == nil {
		return sql.NullBool{Valid: true, Bool: val}
	}
	return 
}
