package query_test

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kod2ulz/gostart/config"
	"github.com/kod2ulz/gostart/query"
)

var _ = Describe("OR Operator Functionality", func() {
	It("should handle OR operations with pipe syntax within fields", func() {
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
			query.Text("username").WithOperators(query.CompareEqual, query.CompareLike),
		)

		ctx := context.Background()
		provider := mockParameterProvider(map[string]string{
			"username": "john|jane|bob", // OR operation
		})

		fmt.Println("=== Testing OR operation ===")
		urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
		conditions := urlParams.GetConditions()

		fmt.Printf("Number of conditions: %d\n", len(conditions))
		for i, cond := range conditions {
			fmt.Printf("Condition %d: %+v, Operator: %v\n", i, cond, cond.Operator)
		}

		Expect(len(conditions)).To(Equal(1))
		Expect(conditions[0].Operator).To(Equal(query.CompareIn))
		Expect(conditions[0].Value).To(Equal([]any{"john", "jane", "bob"}))
	})

	It("should handle literal pipe values with tilde wildcard (no OR processing)", func() {
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
			query.Text("username").WithOperators(query.CompareLike),
		)

		ctx := context.Background()
		provider := mockParameterProvider(map[string]string{
			"~username": "gmail|yahoo|hotmail", // Literal value, not OR operation (tilde OR processing removed)
		})

		fmt.Println("=== Testing literal pipe value with tilde ===")
		urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
		conditions := urlParams.GetConditions()

		fmt.Printf("Number of conditions: %d\n", len(conditions))
		for i, cond := range conditions {
			fmt.Printf("Condition %d: %+v, Operator: %v\n", i, cond, cond.Operator)
		}

		Expect(len(conditions)).To(Equal(1))
		Expect(conditions[0].Operator).To(Equal(query.CompareLike))
		Expect(conditions[0].Value).To(Equal("%gmail|yahoo|hotmail"))
	})
})
