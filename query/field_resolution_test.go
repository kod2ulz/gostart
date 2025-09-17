package query_test

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kod2ulz/gostart/config"
	"github.com/kod2ulz/gostart/query"
)

var _ = Describe("Field Resolution", func() {
	It("should resolve field names with separator variations", func() {
		fmt.Println("=== Testing Field Resolution ===")

		mockParameterProvider := func(params map[string]string) query.UrlParameterProvider {
			return func(ctx context.Context, name string, _default ...string) config.Value {
				fmt.Printf("Query called with: '%s'\n", name)
				if val, ok := params[name]; ok {
					fmt.Printf("Found value: '%s'\n", val)
					return config.Value(val)
				}
				fmt.Println("No value found")
				return ""
			}
		}

		userFieldDefinitions := query.NewDefinitions(
			query.Text("person_id").WithOperators(query.CompareEqual),
		)

		ctx := context.Background()
		provider := mockParameterProvider(map[string]string{
			"personId": "123", // camelCase parameter
		})

		fmt.Println("=== Loading URL parameters ===")
		urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
		conditions := urlParams.GetConditions()

		fmt.Printf("Number of conditions: %d\n", len(conditions))
		for i, cond := range conditions {
			fmt.Printf("Condition %d: %+v\n", i, cond)
		}

		// The test should work with proper field resolution
		Expect(len(conditions)).To(Equal(1))
		Expect(conditions[0].DBPath).To(Equal([]string{"person_id"}))
		Expect(conditions[0].Value).To(Equal("123"))
	})

	It("should handle case-insensitive separator variations", func() {
		fmt.Println("=== Testing Case-Insensitive Separator Variations ===")

		mockParameterProvider := func(params map[string]string) query.UrlParameterProvider {
			return func(ctx context.Context, name string, _default ...string) config.Value {
				fmt.Printf("Query called with: '%s'\n", name)
				if val, ok := params[name]; ok {
					fmt.Printf("Found value: '%s'\n", val)
					return config.Value(val)
				}
				fmt.Println("No value found")
				return ""
			}
		}

		userFieldDefinitions := query.NewDefinitions(
			query.Text("first_name").WithOperators(query.CompareEqual),
		)

		ctx := context.Background()
		provider := mockParameterProvider(map[string]string{
			"firstName": "John", // camelCase parameter for snake_case field
		})

		fmt.Println("=== Loading URL parameters ===")
		urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
		conditions := urlParams.GetConditions()

		fmt.Printf("Number of conditions: %d\n", len(conditions))
		for i, cond := range conditions {
			fmt.Printf("Condition %d: %+v\n", i, cond)
		}

		// Should work with separator variations
		Expect(len(conditions)).To(Equal(1))
		Expect(conditions[0].DBPath).To(Equal([]string{"first_name"}))
		Expect(conditions[0].Value).To(Equal("John"))
	})

	It("should preserve word casing differences", func() {
		fmt.Println("=== Testing Word Casing Preservation ===")

		mockParameterProvider := func(params map[string]string) query.UrlParameterProvider {
			return func(ctx context.Context, name string, _default ...string) config.Value {
				fmt.Printf("Query called with: '%s'\n", name)
				if val, ok := params[name]; ok {
					fmt.Printf("Found value: '%s'\n", val)
					return config.Value(val)
				}
				fmt.Println("No value found")
				return ""
			}
		}

		userFieldDefinitions := query.NewDefinitions(
			query.Text("username").WithOperators(query.CompareEqual),
		)

		ctx := context.Background()
		provider := mockParameterProvider(map[string]string{
			"username": "john", // should match
			"userName": "jane", // should NOT match (different word casing)
		})

		fmt.Println("=== Loading URL parameters ===")
		urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
		conditions := urlParams.GetConditions()

		fmt.Printf("Number of conditions: %d\n", len(conditions))
		for i, cond := range conditions {
			fmt.Printf("Condition %d: %+v\n", i, cond)
		}

		// Should only match the exact field name (username matches, userName does not)
		Expect(len(conditions)).To(Equal(1))
		Expect(conditions[0].DBPath).To(Equal([]string{"username"}))
		Expect(conditions[0].Value).To(Equal("john"))
	})
})
