package query

import (
	"strings"

	"github.com/kod2ulz/gostart/api/openapi"
)

// GenerateOpenAPIParameters creates OpenAPI parameter documentation from field definitions
// This converts your query field definitions into Swagger-compatible parameter documentation
func GenerateOpenAPIParameters(defs FieldDefinitions) []openapi.ParameterAnnotation {
	params := make([]openapi.ParameterAnnotation, 0)

	// Add common search/sort parameters
	params = append(params, getCommonSearchParameters()...)

	// Add field-specific parameters
	for name, def := range defs {
		params = append(params, getFieldParameters(name, def)...)
	}

	return params
}

// getCommonSearchParameters returns standard search/sort/pagination parameters
func getCommonSearchParameters() []openapi.ParameterAnnotation {
	return []openapi.ParameterAnnotation{
		{
			Name:        "sort",
			In:          "query",
			Required:    false,
			Description: "Sort fields (e.g., `-name,+id` for name desc, id asc). Prefix with `-` for descending, `+` or no prefix for ascending.",
			Schema: &openapi.Schema{
				Type:        "string",
				Example:     "-name,+id",
				Description: "Comma-separated list of fields to sort by",
			},
		},
		{
			Name:        "limit",
			In:          "query",
			Required:    false,
			Description: "Number of items to return per page (default: 20)",
			Schema: &openapi.Schema{
				Type:        "integer",
				Format:      "int64",
				Default:     20,
				Minimum:     float64(1),
				Maximum:     float64(100),
			},
		},
		{
			Name:        "offset",
			In:          "query",
			Required:    false,
			Description: "Number of items to skip (default: 0)",
			Schema: &openapi.Schema{
				Type:        "integer",
				Format:      "int64",
				Default:     0,
				Minimum:     0,
			},
		},
		{
			Name:        "page",
			In:          "query",
			Required:    false,
			Description: "Page number (alternative to offset, calculated as (page-1) * limit)",
			Schema: &openapi.Schema{
				Type:        "integer",
				Format:      "int64",
				Default:     1,
				Minimum:     1,
			},
		},
	}
}

// getFieldParameters generates parameters for a specific field based on its definition
func getFieldParameters(name string, def FieldDefinition) []openapi.ParameterAnnotation {
	params := make([]openapi.ParameterAnnotation, 0)

	// Add tilde wildcard parameters for text fields
	for _, op := range def.Operators {
		if op == CompareLike {
			params = append(params, openapi.ParameterAnnotation{
				Name:        "~" + name + "~",
				In:          "query",
				Required:    false,
				Description: "Search " + name + " that contains the value (case-insensitive)",
				Schema: &openapi.Schema{
					Type:        "string",
					Example:     "searchterm",
				},
			})

			params = append(params, openapi.ParameterAnnotation{
				Name:        "~" + name,
				In:          "query",
				Required:    false,
				Description: "Search " + name + " that starts with the value (case-insensitive)",
				Schema: &openapi.Schema{
					Type:        "string",
					Example:     "prefix",
				},
			})

			params = append(params, openapi.ParameterAnnotation{
				Name:        name + "~",
				In:          "query",
				Required:    false,
				Description: "Search " + name + " that ends with the value (case-insensitive)",
				Schema: &openapi.Schema{
					Type:        "string",
					Example:     "suffix",
				},
			})
		}

		if op == CompareEqual {
			params = append(params, openapi.ParameterAnnotation{
				Name:        name,
				In:          "query",
				Required:    false,
				Description: "Exact match for " + name,
				Schema:      getSchemaForType(def.Type),
			})
		}

		if op == CompareGreaterThan {
			params = append(params, openapi.ParameterAnnotation{
				Name:        name + "_gt",
				In:          "query",
				Required:    false,
				Description: name + " greater than value",
				Schema:      getSchemaForType(def.Type),
			})
		}

		if op == CompareLessThan {
			params = append(params, openapi.ParameterAnnotation{
				Name:        name + "_lt",
				In:          "query",
				Required:    false,
				Description: name + " less than value",
				Schema:      getSchemaForType(def.Type),
			})
		}

		if op == CompareGreaterThanOrEqual {
			params = append(params, openapi.ParameterAnnotation{
				Name:        name + "_gte",
				In:          "query",
				Required:    false,
				Description: name + " greater than or equal to value",
				Schema:      getSchemaForType(def.Type),
			})
		}

		if op == CompareLessThanOrEqual {
			params = append(params, openapi.ParameterAnnotation{
				Name:        name + "_lte",
				In:          "query",
				Required:    false,
				Description: name + " less than or equal to value",
				Schema:      getSchemaForType(def.Type),
			})
		}

		if op == CompareBetween {
			params = append(params, openapi.ParameterAnnotation{
				Name:        name + "_bt",
				In:          "query",
				Required:    false,
				Description: name + " between two values, colon-separated (e.g., `1:10` or `2023-01-01:2023-12-31`)",
				Schema:      getSchemaForType(def.Type),
				Example:     "1:10",
			})
		}

		if op == CompareIn {
			params = append(params, openapi.ParameterAnnotation{
				Name:        name + "_in",
				In:          "query",
				Required:    false,
				Description: name + " in list of comma-separated values",
				Schema: &openapi.Schema{
					Type:        "array",
					Items:       getSchemaForType(def.Type),
					Example:     "value1,value2,value3",
				},
			})
		}
	}

	return params
}

// getSchemaForType returns an OpenAPI schema for a given data type
func getSchemaForType(dataType DataType) *openapi.Schema {
	switch dataType {
	case TypeText:
		return &openapi.Schema{
			Type:        "string",
			Description: "Text value",
		}
	case TypeInt:
		return &openapi.Schema{
			Type:        "integer",
			Format:      "int64",
			Description: "Integer value",
		}
	case TypeFloat:
		return &openapi.Schema{
			Type:        "number",
			Format:      "double",
			Description: "Floating point number",
		}
	case TypeBool:
		return &openapi.Schema{
			Type:        "boolean",
			Description: "Boolean value (true/false)",
			Enum:        []interface{}{true, false},
		}
	case TypeDate:
		return &openapi.Schema{
			Type:        "string",
			Format:      "date",
			Description: "Date value (YYYY-MM-DD)",
			Example:     "2023-12-25",
		}
	case TypeTime:
		return &openapi.Schema{
			Type:        "string",
			Format:      "date-time",
			Description: "DateTime value (RFC3339)",
			Example:     "2023-12-25T14:30:00Z",
		}
	case TypeUUID:
		return &openapi.Schema{
			Type:        "string",
			Format:      "uuid",
			Description: "UUID value",
			Example:     "550e8400-e29b-41d4-a716-446655440000",
		}
	case TypeJSON:
		return &openapi.Schema{
			Type:        "object",
			Description: "JSON object",
		}
	default:
		return &openapi.Schema{
			Type:        "string",
			Description: "String value",
		}
	}
}

// GenerateSearchDescription creates a human-readable description of search capabilities
func GenerateSearchDescription(defs FieldDefinitions) string {
	var sb strings.Builder

	sb.WriteString("### Search Parameters\n\n")
	sb.WriteString("This endpoint supports powerful search and filtering capabilities:\n\n")

	sb.WriteString("**Tilde Wildcard Patterns (for text fields):**\n")
	sb.WriteString("- `~field~=value` - Contains the value (ILIKE '%value%')\n")
	sb.WriteString("- `~field=value` - Starts with value (ILIKE '%value')\n")
	sb.WriteString("- `field~=value` - Ends with value (ILIKE 'value%')\n\n")

	sb.WriteString("**Comparison Operators:**\n")
	sb.WriteString("- `field=value` - Exact match\n")
	sb.WriteString("- `field_gt=value` - Greater than\n")
	sb.WriteString("- `field_lt=value` - Less than\n")
	sb.WriteString("- `field_gte=value` - Greater than or equal\n")
	sb.WriteString("- `field_lte=value` - Less than or equal\n")
	sb.WriteString("- `field_bt=value1:value2` - Between two values\n")
	sb.WriteString("- `field_in=value1,value2,value3` - In list\n\n")

	sb.WriteString("**Sorting:**\n")
	sb.WriteString("- `sort=-field1,+field2` - Sort by multiple fields\n")
	sb.WriteString("- Prefix with `-` for descending, `+` or no prefix for ascending\n\n")

	sb.WriteString("**Pagination:**\n")
	sb.WriteString("- `limit=20` - Number of items per page (default: 20)\n")
	sb.WriteString("- `offset=0` - Number of items to skip\n")
	sb.WriteString("- `page=1` - Page number (alternative to offset)\n\n")

	sb.WriteString("**Available Fields:**\n")
	for name, def := range defs {
		sb.WriteString("- **" + name + "** (" + string(def.Type) + ")")
		if def.Sort {
			sb.WriteString(" - *sortable*")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
