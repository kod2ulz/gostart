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

		// Load tilde wildcard syntax for LIKE operations
		if slices.Contains(def.Operators, CompareLike) {
			s.loadTildeWildcard(ctx, apiPath, dbPath, def)
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

// findFieldValue tries to find a field value by checking different separator variations
// Only applies separator variations (., -, _) to field names, not word casing
func (s *urlSearch) findFieldValue(ctx context.Context, fieldName string) config.Value {
	// For tilde wildcard patterns, don't apply variations - be exact
	if strings.HasPrefix(fieldName, "~") || strings.HasSuffix(fieldName, "~") {
		return s.query(ctx, fieldName)
	}

	// For sort parameters, handle the field name part with separator variations
	if strings.HasPrefix(fieldName, "sort_") {
		fieldOnly := strings.TrimPrefix(fieldName, "sort_")
		return s.findFieldValueWithPrefix(ctx, "sort_", fieldOnly)
	}

	// For field names with operators (like field_gt, field_lt), handle the field part
	if strings.Contains(fieldName, "_") {
		parts := strings.Split(fieldName, "_")
		if len(parts) == 2 {
			operator := parts[1]
			fieldPart := parts[0]
			// Check if this is a known operator
			switch operator {
			case "gt", "lt", "gte", "lte", "neq", "bt", "in", "lk":
				return s.findFieldValueWithPrefix(ctx, "", fieldPart+"_"+operator)
			}
		}
	}

	// For plain field names, apply separator variations (but preserve word casing)
	return s.findFieldValueWithSeparatorVariations(ctx, fieldName)
}

// findFieldValueWithPrefix finds a field value with separator variations applied only to the field part
func (s *urlSearch) findFieldValueWithPrefix(ctx context.Context, prefix, fieldName string) config.Value {
	// Try the original field name first
	if val := s.query(ctx, prefix+fieldName); val.Valid() {
		return val
	}

	// Try separator variations while preserving word casing
	return s.findFieldValueWithSeparatorVariations(ctx, prefix+fieldName)
}

// findFieldValueWithSeparatorVariations tries different separator variations
// Rules: firstName != firstname, but first_name == firstName == first-name
// For dotted names: apply variations to each part separately
func (s *urlSearch) findFieldValueWithSeparatorVariations(ctx context.Context, fieldName string) config.Value {
	// Handle dotted names (JSONB fields) - apply variations to each part separately
	if strings.Contains(fieldName, ".") {
		return s.findFieldValueWithDotVariations(ctx, fieldName)
	}

	// Handle simple field names - apply separator variations
	return s.findFieldValueWithSimpleVariations(ctx, fieldName)
}

// findFieldValueWithDotVariations handles field names with dots (JSONB fields)
// Each part of the dot-separated name gets separator variations applied separately
func (s *urlSearch) findFieldValueWithDotVariations(ctx context.Context, fieldName string) config.Value {
	parts := strings.Split(fieldName, ".")
	if len(parts) != 2 {
		return config.Value("")
	}

	columnPart := parts[0]
	fieldPart := parts[1]

	// Generate variations for the column part
	columnVariations := s.generateSeparatorVariations(columnPart)

	// Generate variations for the field part
	fieldVariations := s.generateSeparatorVariations(fieldPart)

	// Try all combinations
	for _, colVar := range columnVariations {
		for _, fieldVar := range fieldVariations {
			combined := colVar + "." + fieldVar
			if val := s.query(ctx, combined); val.Valid() {
				return val
			}
		}
	}

	return config.Value("")
}

// findFieldValueWithSimpleVariations handles field names without dots
func (s *urlSearch) findFieldValueWithSimpleVariations(ctx context.Context, fieldName string) config.Value {
	// Try all separator variations
	variations := s.generateSeparatorVariations(fieldName)
	for _, variation := range variations {
		if val := s.query(ctx, variation); val.Valid() {
			return val
		}
	}

	return config.Value("")
}

// generateSeparatorVariations generates all valid separator variations for a field name
// Note: firstName != firstname (word casing is preserved), but first_name == firstName
func (s *urlSearch) generateSeparatorVariations(fieldName string) []string {
	variations := []string{fieldName}

	// Convert to camelCase (if not already)
	if strings.Contains(fieldName, "_") || strings.Contains(fieldName, "-") {
		camelVersion := strcase.ToCamel(fieldName)
		if camelVersion != fieldName {
			variations = append(variations, camelVersion)
		}
	}

	// Convert underscores to hyphens
	if strings.Contains(fieldName, "_") {
		hyphenVersion := strings.ReplaceAll(fieldName, "_", "-")
		if hyphenVersion != fieldName {
			variations = append(variations, hyphenVersion)
		}
	}

	// Convert hyphens to underscores
	if strings.Contains(fieldName, "-") {
		underscoreVersion := strings.ReplaceAll(fieldName, "-", "_")
		if underscoreVersion != fieldName {
			variations = append(variations, underscoreVersion)
		}
	}

	return variations
}

func (s *urlSearch) loadSort(ctx context.Context, apiPath, dbName string) {
	// Try the new format: sort_field=desc
	val := s.findFieldValue(ctx, "sort_"+apiPath)
	if val.Valid() && sortTypeValid(val.String()) {
		s.sorts = append(s.sorts, ParsedSort{
			DBName: dbName,
			Type:   SortType(val.String()),
		})
	}

	// Try legacy format: sort=-field,+field
	if len(s.sorts) == 0 {
		s.loadLegacySort(ctx, apiPath, dbName)
	}
}

func (s *urlSearch) loadLegacySort(ctx context.Context, apiPath, dbName string) {
	// Legacy format: sort=-field,+field (comma-separated list)
	if sortParam := s.query(ctx, "sort"); sortParam.Valid() {
		sortFields := strings.Split(sortParam.String(), ",")
		for _, sortField := range sortFields {
			sortField = strings.TrimSpace(sortField)
			if sortField == "" {
				continue
			}

			// Extract field name and direction
			var fieldName string
			var sortType SortType

			if strings.HasPrefix(sortField, "-") {
				fieldName = sortField[1:]
				sortType = SortDesc
			} else if strings.HasPrefix(sortField, "+") {
				fieldName = sortField[1:]
				sortType = SortAsc
			} else {
				fieldName = sortField
				sortType = SortAsc
			}

			// Check if this field matches our apiPath (case-insensitive)
			if strings.EqualFold(fieldName, apiPath) || strings.EqualFold(fieldName, strcase.ToCamel(apiPath)) {
				s.sorts = append(s.sorts, ParsedSort{
					DBName: dbName,
					Type:   sortType,
				})
			}
		}
	}
}

func (s *urlSearch) loadComparison(ctx context.Context, apiPath string, dbPath []string, def FieldDefinition, op CompareOperator) {
	paramName := fmt.Sprintf("%s_%s", apiPath, string(op))
	if op == CompareEqual { // Allow for shorthand `field=value` for equality
		paramName = apiPath
	}

	val := s.findFieldValue(ctx, paramName)
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
	val := s.findFieldValue(ctx, paramName)
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

func (s *urlSearch) loadTildeWildcard(ctx context.Context, apiPath string, dbPath []string, def FieldDefinition) {
	// Check for tilde wildcard patterns: ~field~, ~field, field~
	// ~field~=value -> field ILIKE '%value%'
	// ~field=value -> field ILIKE '%value'
	// field~=value -> field ILIKE 'value%'

	// Pattern 1: ~field~=value (contains)
	paramName1 := "~" + apiPath + "~"
	if val1 := s.findFieldValue(ctx, paramName1); val1.Valid() {
		if parsedVal, ok := s.parseValue(val1.String(), def); ok {
			s.conditions = append(s.conditions, ParsedCondition{
				DBPath:   dbPath,
				Operator: CompareLike,
				Value:    "%" + fmt.Sprintf("%v", parsedVal) + "%",
			})
		}
	}

	// Pattern 2: ~field=value (starts with)
	paramName2 := "~" + apiPath
	if val2 := s.findFieldValue(ctx, paramName2); val2.Valid() {
		if parsedVal, ok := s.parseValue(val2.String(), def); ok {
			s.conditions = append(s.conditions, ParsedCondition{
				DBPath:   dbPath,
				Operator: CompareLike,
				Value:    "%" + fmt.Sprintf("%v", parsedVal),
			})
		}
	}

	// Pattern 3: field~=value (ends with)
	paramName3 := apiPath + "~"
	if val3 := s.findFieldValue(ctx, paramName3); val3.Valid() {
		if parsedVal, ok := s.parseValue(val3.String(), def); ok {
			s.conditions = append(s.conditions, ParsedCondition{
				DBPath:   dbPath,
				Operator: CompareLike,
				Value:    fmt.Sprintf("%v", parsedVal) + "%",
			})
		}
	}
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
