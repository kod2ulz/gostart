package query_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kod2ulz/gostart/config"
	"github.com/kod2ulz/gostart/query"
)

var _ = Describe("Tilde Wildcard Syntax for LIKE Operations", func() {
	var mockParameterProvider func(params map[string]string) query.UrlParameterProvider
	var userFieldDefinitions query.FieldDefinitions

	BeforeEach(func() {
		mockParameterProvider = func(params map[string]string) query.UrlParameterProvider {
			return func(ctx context.Context, name string, _default ...string) config.Value {
				if val, ok := params[name]; ok {
					return config.Value(val)
				}
				if len(_default) > 0 {
					return config.Value(_default[0])
				}
				return ""
			}
		}

		userFieldDefinitions = query.NewDefinitions(
			query.Text("first_name").Sortable().WithOperators(
				query.CompareEqual, query.CompareLike, query.CompareIn,
			),
			query.Text("email").WithOperators(query.CompareEqual, query.CompareLike),
			query.Int("age").WithOperators(
				query.CompareEqual, query.CompareGreaterThan,
				query.CompareLessThan, query.CompareBetween,
			),
			query.Bool("active"),
			query.JSON("metadata").WithSchema(
				query.Text("city"),
				query.Text("country"),
				query.Int("zipCode"),
			),
		)
	})

	Describe("Tilde Wildcard Patterns", func() {
		It("should handle contains pattern (~field~=value)", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"~first_name~": "john",
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			conditions := urlParams.GetConditions()

			Expect(conditions).To(HaveLen(1))
			Expect(conditions[0].Operator).To(Equal(query.CompareLike))
			Expect(conditions[0].Value).To(Equal("%john%"))
		})

		It("should handle starts with pattern (~field=value)", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"~first_name": "john",
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			conditions := urlParams.GetConditions()

			Expect(conditions).To(HaveLen(1))
			Expect(conditions[0].Operator).To(Equal(query.CompareLike))
			Expect(conditions[0].Value).To(Equal("%john"))
		})

		It("should handle ends with pattern (field~=value)", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"first_name~": "john",
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			conditions := urlParams.GetConditions()

			Expect(conditions).To(HaveLen(1))
			Expect(conditions[0].Operator).To(Equal(query.CompareLike))
			Expect(conditions[0].Value).To(Equal("john%"))
		})

		It("should handle multiple tilde patterns", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"~first_name~": "john",
				"~email":      "example.com",
				"age_gt":      "25",
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			conditions := urlParams.GetConditions()

			Expect(conditions).To(HaveLen(3))

			// Check contains pattern
			Expect(conditions[0].Operator).To(Equal(query.CompareLike))
			Expect(conditions[0].Value).To(Equal("%john%"))

			// Check starts with pattern
			Expect(conditions[1].Operator).To(Equal(query.CompareLike))
			Expect(conditions[1].Value).To(Equal("%example.com"))

			// Check regular comparison
			Expect(conditions[2].Operator).To(Equal(query.CompareGreaterThan))
			Expect(conditions[2].Value).To(Equal(int64(25)))
		})

		It("should ignore tilde patterns for fields that don't support LIKE", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"~age~": "25",
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			conditions := urlParams.GetConditions()

			// Age field doesn't support LIKE, so tilde patterns should be ignored
			Expect(conditions).To(BeEmpty())
		})
	})

	Describe("Case-Insensitive Field Name Resolution", func() {
		It("should handle camelCase field names with separators", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"firstName": "John", // camelCase version of first_name
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			conditions := urlParams.GetConditions()

			Expect(conditions).To(HaveLen(1))
			Expect(conditions[0].Operator).To(Equal(query.CompareEqual))
			Expect(conditions[0].Value).To(Equal("John"))
		})

		It("should handle kebab-case field names", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"first-name": "Jane", // kebab-case version
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			conditions := urlParams.GetConditions()

			Expect(conditions).To(HaveLen(1))
			Expect(conditions[0].Operator).To(Equal(query.CompareEqual))
			Expect(conditions[0].Value).To(Equal("Jane"))
		})

		It("should handle snake_case field names", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"first_name": "Bob", // exact match
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			conditions := urlParams.GetConditions()

			Expect(conditions).To(HaveLen(1))
			Expect(conditions[0].Operator).To(Equal(query.CompareEqual))
			Expect(conditions[0].Value).To(Equal("Bob"))
		})

		It("should NOT match different word casings", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"firstname": "john", // different from first_name (word casing)
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			conditions := urlParams.GetConditions()

			// Should not match since first_name != firstname (different word casing)
			Expect(conditions).To(HaveLen(0))
		})

		It("should handle JSONB field names with dots", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"person.first_name": "John", // JSONB field with dot
			})

			// Create field definitions with JSONB field
			jsonFieldDefinitions := query.NewDefinitions(
				query.JSON("person").WithSchema(
					query.Text("first_name").WithOperators(query.CompareEqual),
				).WithOperators(query.CompareEqual),
			)

			urlParams := query.SearchURL(provider, jsonFieldDefinitions).Load(ctx)
			conditions := urlParams.GetConditions()

			Expect(conditions).To(HaveLen(1))
			Expect(conditions[0].Operator).To(Equal(query.CompareEqual))
			Expect(conditions[0].Value).To(Equal("John"))
		})

		It("should handle JSONB field with camelCase parameter", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"person.firstName": "Jane", // camelCase version should match
			})

			// Create field definitions with JSONB field
			jsonFieldDefinitions := query.NewDefinitions(
				query.JSON("person").WithSchema(
					query.Text("first_name").WithOperators(query.CompareEqual),
				).WithOperators(query.CompareEqual),
			)

			urlParams := query.SearchURL(provider, jsonFieldDefinitions).Load(ctx)
			conditions := urlParams.GetConditions()

			Expect(conditions).To(HaveLen(1))
			Expect(conditions[0].Operator).To(Equal(query.CompareEqual))
			Expect(conditions[0].Value).To(Equal("Jane"))
		})

		It("should handle mixed case operators", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"userName_gt": "25", // camelCase with operator
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			conditions := urlParams.GetConditions()

			Expect(conditions).To(HaveLen(1))
			Expect(conditions[0].Operator).To(Equal(query.CompareGreaterThan))
			Expect(conditions[0].Value).To(Equal(int64(25)))
		})

		It("should handle JSON field case variations", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"metadata.city":     "New York",
				"metadata_zip-code": "10001", // kebab-case for JSON field
				"metadata_country":  "USA",     // snake_case for JSON field
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			conditions := urlParams.GetConditions()

			Expect(conditions).To(HaveLen(3))
		})
	})

	Describe("Sort Parameter Format Support", func() {
		It("should handle new format sort_field=desc", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"sort_first_name": "desc",
				"sort_age":      "asc",
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			sorts := urlParams.GetSorts()

			Expect(sorts).To(HaveLen(2))
			Expect(sorts[0].DBName).To(Equal("person_id"))
			Expect(sorts[0].Type).To(Equal(query.SortDesc))
			Expect(sorts[1].DBName).To(Equal("age"))
			Expect(sorts[1].Type).To(Equal(query.SortAsc))
		})

		It("should handle legacy format sort=-field,+field", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"sort": "-person_id,+age",
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			sorts := urlParams.GetSorts()

			Expect(sorts).To(HaveLen(2))
			Expect(sorts[0].DBName).To(Equal("person_id"))
			Expect(sorts[0].Type).To(Equal(query.SortDesc))
			Expect(sorts[1].DBName).To(Equal("age"))
			Expect(sorts[1].Type).To(Equal(query.SortAsc))
		})

		It("should handle legacy format without explicit + sign", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"sort": "-person_id,age",
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			sorts := urlParams.GetSorts()

			Expect(sorts).To(HaveLen(2))
			Expect(sorts[0].DBName).To(Equal("person_id"))
			Expect(sorts[0].Type).To(Equal(query.SortDesc))
			Expect(sorts[1].DBName).To(Equal("age"))
			Expect(sorts[1].Type).To(Equal(query.SortAsc))
		})

		It("should prefer new format over legacy format", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"sort_first_name": "desc", // New format
				"sort":           "-person_id,+age", // Legacy format
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			sorts := urlParams.GetSorts()

			// Should only use new format
			Expect(sorts).To(HaveLen(1))
			Expect(sorts[0].DBName).To(Equal("person_id"))
			Expect(sorts[0].Type).To(Equal(query.SortDesc))
		})

		It("should handle case-insensitive sort field names in legacy format", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"sort": "-UserName,+Age", // Mixed case
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			sorts := urlParams.GetSorts()

			Expect(sorts).To(HaveLen(2))
			Expect(sorts[0].DBName).To(Equal("person_id"))
			Expect(sorts[0].Type).To(Equal(query.SortDesc))
			Expect(sorts[1].DBName).To(Equal("age"))
			Expect(sorts[1].Type).To(Equal(query.SortAsc))
		})
	})

	Describe("Integration with Other Features", func() {
		It("should work with parameter overrides", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"~username~": "john",
				"age_gt":     "25",
			})

			var urlParams query.URLSearchParam = query.SearchURL(provider, userFieldDefinitions).Load(ctx)

			// Add service layer constraint
			urlParams = urlParams.AddField("active", true)

			conditions := urlParams.GetConditions()
			Expect(conditions).To(HaveLen(3))

			// Check that all conditions are present
			Expect(conditions[0].Operator).To(Equal(query.CompareLike))
			Expect(conditions[0].Value).To(Equal("%john%"))

			Expect(conditions[1].Operator).To(Equal(query.CompareGreaterThan))
			Expect(conditions[1].Value).To(Equal(int64(25)))

			Expect(conditions[2].Operator).To(Equal(query.CompareEqual))
			Expect(conditions[2].Value).To(Equal(true))
		})

		It("should generate correct SQL for tilde wildcards", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"~username~": "john",
				"~email":     "example.com",
				"sort":       "-first_name",
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			sqlQuery, args := qb.Criteria()

			Expect(sqlQuery.String()).To(ContainSubstring("first_name ilike"))
			Expect(sqlQuery.String()).To(ContainSubstring("email ilike"))
			Expect(sqlQuery.String()).To(ContainSubstring("order by first_name desc"))
			Expect(len(args)).To(Equal(2))
			Expect(args[0]).To(Equal("%john%"))
			Expect(args[1]).To(Equal("%example.com"))
		})
	})

	Describe("Edge Cases and Error Handling", func() {
		It("should handle empty tilde parameters", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"~username~": "",
				"~email":     "",
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			conditions := urlParams.GetConditions()

			// Empty values should be processed but may be filtered out by business logic
			Expect(conditions).To(HaveLen(2))
		})

		It("should handle malformed legacy sort parameters", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"sort": "-,,+person_id,,age",
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			sorts := urlParams.GetSorts()

			// Should skip empty fields
			Expect(sorts).To(HaveLen(2))
			Expect(sorts[0].DBName).To(Equal("person_id"))
			Expect(sorts[1].DBName).To(Equal("age"))
		})

		It("should handle unknown field names gracefully", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"~unknown_field~": "value",
				"sort_unknown":    "desc",
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			conditions := urlParams.GetConditions()
			sorts := urlParams.GetSorts()

			// Unknown fields should be ignored
			Expect(conditions).To(BeEmpty())
			Expect(sorts).To(BeEmpty())
		})
	})
})