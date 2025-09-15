package query

import (
	"github.com/iancoleman/strcase"
)

// DataType defines the type of data a field represents.
type DataType string

const (
	TypeText    DataType = "text"
	TypeInt     DataType = "int"
	TypeFloat   DataType = "float"
	TypeDate    DataType = "date"
	TypeTime    DataType = "time"
	TypeUUID    DataType = "uuid"
	TypeJSON    DataType = "json"
	TypeBool    DataType = "bool"
)

// FieldDefinition holds the complete definition for a queryable field.
type FieldDefinition struct {
	Name       string
	DBName     string
	Type       DataType
	TimeFormat string
	Operators  []CompareOperator
	Sort       bool
	Schema     FieldDefinitions
}

// WithDBName overrides the default snake_case database column name.
func (fd FieldDefinition) WithDBName(name string) FieldDefinition {
	fd.DBName = name
	return fd
}

// Sortable enables or disables sorting on this field.
// Defaults to true if no argument is provided.
func (fd FieldDefinition) Sortable(isSortable ...bool) FieldDefinition {
	fd.Sort = len(isSortable) == 0 || isSortable[0]
	return fd
}

// WithOperators sets the list of allowed comparison operators for this field.
func (fd FieldDefinition) WithOperators(operators ...CompareOperator) FieldDefinition {
	fd.Operators = operators
	return fd
}

// WithSchema defines the nested queryable fields for a JSON type.
func (fd FieldDefinition) WithSchema(fields ...FieldDefinition) FieldDefinition {
	if fd.Type != TypeJSON {
		return fd // Silently ignore
	}
	if fd.Schema == nil {
		fd.Schema = make(FieldDefinitions)
	}
	for _, field := range fields {
		fd.Schema[field.Name] = field
	}
	return fd
}

// FieldDefinitions is a map of API-facing field names to their definitions.
type FieldDefinitions map[string]FieldDefinition

// NewDefinitions creates a new FieldDefinitions map, optionally initialized with a set of fields.
func NewDefinitions(fields ...FieldDefinition) FieldDefinitions {
	defs := make(FieldDefinitions)
	return defs.Add(fields...)
}

// Add appends one or more FieldDefinition to the map.
func (defs FieldDefinitions) Add(fields ...FieldDefinition) FieldDefinitions {
	for _, field := range fields {
		defs[field.Name] = field
	}
	return defs
}

// --- Builder Functions ---

func newField(name string, dataType DataType) FieldDefinition {
	return FieldDefinition{
		Name:      name,
		DBName:    strcase.ToSnake(name),
		Type:      dataType,
		Sort:      false,
		Operators: make([]CompareOperator, 0),
	}
}

// Text defines a field of type string.
func Text(name string) FieldDefinition {
	return newField(name, TypeText).WithOperators(CompareEqual, CompareNotEqual, CompareIn, CompareLike)
}

// Int defines a field of type integer.
func Int(name string) FieldDefinition {
	return newField(name, TypeInt).WithOperators(CompareEqual, CompareNotEqual, CompareIn, CompareBetween, CompareGreaterThan, CompareGreaterThanOrEqual, CompareLessThan, CompareLessThanOrEqual)
}

// Float defines a field of type float.
func Float(name string) FieldDefinition {
	return newField(name, TypeFloat).WithOperators(CompareEqual, CompareNotEqual, CompareIn, CompareBetween, CompareGreaterThan, CompareGreaterThanOrEqual, CompareLessThan, CompareLessThanOrEqual)
}

// Date defines a field of type date.
func Date(name string, format ...string) FieldDefinition {
	fd := newField(name, TypeDate).WithOperators(CompareEqual, CompareNotEqual, CompareBetween, CompareGreaterThan, CompareGreaterThanOrEqual, CompareLessThan, CompareLessThanOrEqual)
	if len(format) > 0 {
		fd.TimeFormat = format[0]
	}
	return fd
}

// Time defines a field of type timestamp.
func Time(name string, format ...string) FieldDefinition {
	fd := newField(name, TypeTime).WithOperators(CompareEqual, CompareNotEqual, CompareBetween, CompareGreaterThan, CompareGreaterThanOrEqual, CompareLessThan, CompareLessThanOrEqual)
	if len(format) > 0 {
		fd.TimeFormat = format[0]
	}
	return fd
}

// UUID defines a field of type UUID.
func UUID(name string) FieldDefinition {
	return newField(name, TypeUUID).WithOperators(CompareEqual, CompareNotEqual, CompareIn)
}

// Bool defines a field of type boolean.
func Bool(name string) FieldDefinition {
	return newField(name, TypeBool).WithOperators(CompareEqual, CompareNotEqual)
}

// JSON defines a field of type JSON.
func JSON(name string) FieldDefinition {
	return newField(name, TypeJSON)
}
