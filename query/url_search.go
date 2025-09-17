package query

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/iancoleman/strcase"
	"github.com/kod2ulz/gostart/config"
)

// UrlParameterProvider is a function type that reads a value from a URL query.
type UrlParameterProvider func(ctx context.Context, name string, _default ...string) (out config.Value)

// ParsedCondition holds a validated and typed condition ready for the SQL builder.
type ParsedCondition struct {
	DBPath   []string
	Operator CompareOperator
	Value    any
}

// ParsedSort holds a validated sort instruction.
type ParsedSort struct {
	DBName string
	Type   SortType
}

// URLSearchParam is the interface for accessing parsed URL search parameters.
type URLSearchParam interface {
	GetConditions() []ParsedCondition
	GetSorts() []ParsedSort
	GetLimit() int64
	GetOffset() int64

	// Parameter override methods for service/data layer
	AddCondition(field string, operator CompareOperator, value any) URLSearchParam
	AddField(field string, value any) URLSearchParam
	AddSort(field string, sortType SortType) URLSearchParam
	SetLimit(limit int64) URLSearchParam
	SetOffset(offset int64) URLSearchParam
}

// SearchURL initializes a new URL search parser.
func SearchURL(queryReader UrlParameterProvider, defs FieldDefinitions) *urlSearch {
	return &urlSearch{
		query:      queryReader,
		defs:       defs,
		conditions: make([]ParsedCondition, 0),
		sorts:      make([]ParsedSort, 0),
	}
}

type urlSearch struct {
	limit      int64
	offset     int64
	conditions []ParsedCondition
	sorts      []ParsedSort
	query      UrlParameterProvider
	defs       FieldDefinitions
}

// Load parses the URL query parameters based on the provided field definitions.
func (s *urlSearch) Load(ctx context.Context) *urlSearch {
	s.loadBoundaries(ctx)
	s.loadFields(ctx, s.defs, []string{}, "")
	return s
}

func (s *urlSearch) loadBoundaries(ctx context.Context) {
	s.limit = s.query(ctx, "limit", fmt.Sprint(SELECT_LIMIT)).Int64()
	s.offset = s.query(ctx, "offset", "0").Int64()
	if s.offset == 0 {
		if page := s.query(ctx, "page", "0").Int64(); page > 1 {
			s.offset = (page - 1) * s.limit
		}
	}
}

func (s *urlSearch) loadFields(ctx context.Context, defs FieldDefinitions, parentDBPath []string, parentAPIPath string) {
	for name, def := range defs {
		apiPath := name
		if parentAPIPath != "" {
			apiPath = parentAPIPath + "." + name
		}

		dbPath := append(parentDBPath, def.DBName)

		if def.Type == TypeJSON && def.Schema != nil {
			s.loadFields(ctx, def.Schema, dbPath, apiPath)
		}

		if def.Sort {
			s.loadSort(ctx, apiPath, def.DBName)
		}

		for _, op := range def.Operators {
			if op == CompareBetween { // Skip between, as it's handled separately
				continue
			}
			s.loadComparison(ctx, apiPath, dbPath, def, op)
		}

		// Special handling for 'between' operator
		if hasBetween := slices.Contains(def.Operators, CompareBetween); hasBetween {
			s.loadBetweenComparison(ctx, apiPath, dbPath, def)
		}
	}
}

func (s *urlSearch) loadSort(ctx context.Context, apiPath, dbName string) {
	val := s.query(ctx, "sort_"+apiPath)
	if !val.Valid() {
		val = s.query(ctx, "sort_"+strcase.ToCamel(apiPath))
	}
	if val.Valid() && sortTypeValid(val.String()) {
		s.sorts = append(s.sorts, ParsedSort{
			DBName: dbName,
			Type:   SortType(val.String()),
		})
	}
}

func (s *urlSearch) loadComparison(ctx context.Context, apiPath string, dbPath []string, def FieldDefinition, op CompareOperator) {
	paramName := fmt.Sprintf("%s_%s", apiPath, string(op))
	if op == CompareEqual { // Allow for shorthand `field=value` for equality
		paramName = apiPath
	}

	val := s.query(ctx, paramName)
	if !val.Valid() {
		val = s.query(ctx, strcase.ToCamel(paramName))
	}
	if !val.Valid() {
		return
	}

	parsedVal, ok := s.parseValue(val.String(), def)
	if !ok {
		return // Skip if parsing fails
	}

	s.conditions = append(s.conditions, ParsedCondition{
		DBPath:   dbPath,
		Operator: op,
		Value:    parsedVal,
	})
}

func (s *urlSearch) loadBetweenComparison(ctx context.Context, apiPath string, dbPath []string, def FieldDefinition) {
	paramName := fmt.Sprintf("%s_%s", apiPath, string(CompareBetween))
	val := s.query(ctx, paramName)
	if !val.Valid() {
		val = s.query(ctx, strcase.ToCamel(paramName))
	}
	if !val.Valid() {
		return
	}

	parts := strings.Split(val.String(), ":")
	if len(parts) != 2 {
		return // Invalid format for between
	}

	from, ok1 := s.parseValue(parts[0], def)
	to, ok2 := s.parseValue(parts[1], def)

	if !ok1 || !ok2 {
		return // Failed to parse one of the values
	}

	s.conditions = append(s.conditions, ParsedCondition{
		DBPath:   dbPath,
		Operator: CompareBetween,
		Value:    []any{from, to},
	})
}

func (s *urlSearch) parseValue(val string, def FieldDefinition) (any, bool) {
	switch def.Type {
	case TypeText, TypeUUID:
		return val, true
	case TypeBool:
		b, err := strconv.ParseBool(val)
		return b, err == nil
	case TypeInt:
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return i, true
		}
	case TypeFloat:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f, true
		}
	case TypeDate, TypeTime:
		format := def.TimeFormat
		if format == "" {
			format = time.RFC3339
		}
		if t, err := time.Parse(format, val); err == nil {
			return t, true
		}
	}
	return nil, false
}

// GetConditions returns the parsed and validated query conditions.
func (s *urlSearch) GetConditions() []ParsedCondition {
	return s.conditions
}

// GetSorts returns the parsed and validated sort instructions.
func (s *urlSearch) GetSorts() []ParsedSort {
	return s.sorts
}

// GetLimit returns the pagination limit.
func (s *urlSearch) GetLimit() int64 {
	return s.limit
}

// GetOffset returns the pagination offset.
func (s *urlSearch) GetOffset() int64 {
	return s.offset
}

// AddCondition adds a new condition to the search parameters.
func (s *urlSearch) AddCondition(field string, operator CompareOperator, value any) URLSearchParam {
	s.conditions = append(s.conditions, ParsedCondition{
		DBPath:   []string{field},
		Operator: operator,
		Value:    value,
	})
	return s
}

// AddField adds a new field equality condition.
func (s *urlSearch) AddField(field string, value any) URLSearchParam {
	return s.AddCondition(field, CompareEqual, value)
}

// AddSort adds a new sort instruction.
func (s *urlSearch) AddSort(field string, sortType SortType) URLSearchParam {
	s.sorts = append(s.sorts, ParsedSort{
		DBName: field,
		Type:   sortType,
	})
	return s
}

// SetLimit sets the pagination limit.
func (s *urlSearch) SetLimit(limit int64) URLSearchParam {
	s.limit = limit
	return s
}

// SetOffset sets the pagination offset.
func (s *urlSearch) SetOffset(offset int64) URLSearchParam {
	s.offset = offset
	return s
}
