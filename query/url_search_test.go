package query_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/query"
)

var _ = Describe("URL Search", func() {

	var mockProvider query.UrlParameterProvider
	var defs query.FieldDefinitions
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		mockProvider = func(ctx context.Context, name string, _default ...string) contracts.Value {
			// Return empty value by default
			if len(_default) > 0 {
				return contracts.Value(_default[0])
			}
			return ""
		}

		defs = query.NewDefinitions(
			query.Text("name"),
			query.Int("age"),
			query.Float("score"),
			query.Date("birth_date", "2006-01-02"),
			query.Bool("active"),
			query.JSON("metadata").WithSchema(
				query.Text("city"),
				query.Int("zip_code"),
			),
		)
	})

	Describe("SearchURL", func() {
		It("should create new URL search instance", func() {
			search := query.SearchURL(mockProvider, defs)
			Expect(search).NotTo(BeNil())
		})

		It("should handle empty field definitions", func() {
			emptyDefs := query.NewDefinitions()
			search := query.SearchURL(mockProvider, emptyDefs)
			Expect(search).NotTo(BeNil())
		})

		It("should load parameters and return URLSearchParam interface", func() {
			mockProvider = func(ctx context.Context, name string, _default ...string) contracts.Value {
				switch name {
				case "name":
					return contracts.Value("John Doe")
				case "age_gt":
					return contracts.Value("25")
				case "sort":
					return contracts.Value("-name")
				case "limit":
					return contracts.Value("10")
				default:
					return contracts.Value("")
				}
			}

			search := query.SearchURL(mockProvider, defs)
			params := search.Load(ctx)

			// Verify it implements the interface
			Expect(params).NotTo(BeNil())

			// Test interface methods
			conditions := params.GetConditions()
			Expect(conditions).NotTo(BeNil())

			sorts := params.GetSorts()
			Expect(sorts).NotTo(BeNil())

			Expect(params.GetLimit()).To(BeNumerically(">", 0))
		})
	})

	Describe("Real-world Scenarios", func() {
		It("should handle user search scenario", func() {
			userDefs := query.NewDefinitions(
				query.Text("username").Sortable().WithOperators(
					query.CompareEqual, query.CompareLike, query.CompareIn,
				),
				query.Text("email").WithOperators(query.CompareEqual, query.CompareIn),
				query.Int("age").WithOperators(
					query.CompareEqual, query.CompareGreaterThan,
					query.CompareLessThan, query.CompareBetween,
				),
				query.Bool("active"),
				query.Date("created_at", time.RFC3339).Sortable(),
			)

			mockProvider = func(ctx context.Context, name string, _default ...string) contracts.Value {
				switch name {
				case "username_lyk":
					return contracts.Value("john%")
				case "age_bt":
					return contracts.Value("25:35")
				case "active":
					return contracts.Value("true")
				case "sort":
					return contracts.Value("-created_at,username")
				case "limit":
					return contracts.Value("50")
				default:
					return contracts.Value("")
				}
			}

			search := query.SearchURL(mockProvider, userDefs)
			params := search.Load(ctx)

			Expect(params).NotTo(BeNil())
			Expect(params.GetLimit()).To(Equal(int64(50)))
		})
	})
})
