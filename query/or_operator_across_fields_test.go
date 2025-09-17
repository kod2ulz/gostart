package query_test

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kod2ulz/gostart/config"
	"github.com/kod2ulz/gostart/query"
)

var _ = Describe("OR Operator Across Fields", func() {
	It("should handle OR operations with star prefix syntax across fields", func() {
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
			query.Text("role").WithOperators(query.CompareEqual),
			query.Text("accountId").WithOperators(query.CompareEqual),
		)

		ctx := context.Background()
		provider := mockParameterProvider(map[string]string{
			"*role":     "admin", // Star-prefixed field 1
			"accountId": "123",   // Regular field
			"*foo":      "bar",   // Star-prefixed field 2 (but foo is not defined)
		})

		fmt.Println("=== Testing OR operation across fields ===")
		urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
		conditions := urlParams.GetConditions()

		fmt.Printf("Number of conditions: %d\n", len(conditions))
		for i, cond := range conditions {
			fmt.Printf("Condition %d: %+v, Operator: %v\n", i, cond, cond.Operator)
		}

		// Should have 2 conditions: accountId=123 AND *role=admin (both should be present)
		Expect(len(conditions)).To(Equal(2))
		// Find the regular field condition
		var accountCondition, roleCondition *query.ParsedCondition
		for i := range conditions {
			if conditions[i].Operator == query.CompareEqual {
				accountCondition = &conditions[i]
			} else if conditions[i].Operator == query.CompareOr {
				roleCondition = &conditions[i]
			}
		}
		Expect(accountCondition).NotTo(BeNil())
		Expect(accountCondition.DBPath).To(Equal([]string{"account_id"}))
		Expect(accountCondition.Value).To(Equal("123"))
		Expect(roleCondition).NotTo(BeNil())
		Expect(roleCondition.DBPath).To(Equal([]string{"role"}))
		Expect(roleCondition.Value).To(Equal("admin"))
	})

	It("should create OR group only when multiple star fields are present", func() {
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
			query.Text("role").WithOperators(query.CompareEqual),
			query.Text("accountId").WithOperators(query.CompareEqual),
		)

		ctx := context.Background()
		provider := mockParameterProvider(map[string]string{
			"*role":      "admin", // Star-prefixed field 1
			"*accountId": "123",   // Star-prefixed field 2
		})

		fmt.Println("=== Testing OR group with multiple star fields ===")
		urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
		conditions := urlParams.GetConditions()

		fmt.Printf("Number of conditions: %d\n", len(conditions))
		for i, cond := range conditions {
			fmt.Printf("Condition %d: %+v, Operator: %v\n", i, cond, cond.Operator)
		}

		// Should have 2 conditions: both as OR conditions (SQL builder will combine them)
		Expect(len(conditions)).To(Equal(2))
		Expect(conditions[0].Operator).To(Equal(query.CompareOr))
		Expect(conditions[1].Operator).To(Equal(query.CompareOr))
	})

	It("should not create OR group when only one star field is present", func() {
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
			query.Text("role").WithOperators(query.CompareEqual),
			query.Text("accountId").WithOperators(query.CompareEqual),
		)

		ctx := context.Background()
		provider := mockParameterProvider(map[string]string{
			"*role":     "admin", // Only one star-prefixed field
			"accountId": "123",   // Regular field
		})

		fmt.Println("=== Testing single star field (should not create OR group) ===")
		urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
		conditions := urlParams.GetConditions()

		fmt.Printf("Number of conditions: %d\n", len(conditions))
		for i, cond := range conditions {
			fmt.Printf("Condition %d: %+v, Operator: %v\n", i, cond, cond.Operator)
		}

		// Should have 2 conditions: accountId=123 AND *role=admin (both should be present)
		Expect(len(conditions)).To(Equal(2))
		// Find the regular field condition
		var accountCondition, roleCondition *query.ParsedCondition
		for i := range conditions {
			if conditions[i].Operator == query.CompareEqual {
				accountCondition = &conditions[i]
			} else if conditions[i].Operator == query.CompareOr {
				roleCondition = &conditions[i]
			}
		}
		Expect(accountCondition).NotTo(BeNil())
		Expect(accountCondition.DBPath).To(Equal([]string{"account_id"}))
		Expect(accountCondition.Value).To(Equal("123"))
		Expect(roleCondition).NotTo(BeNil())
		Expect(roleCondition.DBPath).To(Equal([]string{"role"}))
		Expect(roleCondition.Value).To(Equal("admin"))
	})

	It("should remove single OR condition when it's the only condition", func() {
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
			query.Text("role").WithOperators(query.CompareEqual),
			query.Text("accountId").WithOperators(query.CompareEqual),
		)

		ctx := context.Background()
		provider := mockParameterProvider(map[string]string{
			"*role": "admin", // Only one star field - should be removed
		})

		fmt.Println("=== Testing single star field removal ===")
		urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
		conditions := urlParams.GetConditions()

		fmt.Printf("Number of conditions: %d\n", len(conditions))
		for i, cond := range conditions {
			fmt.Printf("Condition %d: %+v, Operator: %v\n", i, cond, cond.Operator)
		}

		// Should have 0 conditions - the single OR condition should be removed
		Expect(len(conditions)).To(Equal(0))
	})
})
