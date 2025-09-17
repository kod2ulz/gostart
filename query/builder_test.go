package query_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kod2ulz/gostart/config"
	"github.com/kod2ulz/gostart/query"
)

var _ = Describe("SQL Builder", func() {

	var mockParameterProvider func(params map[string]string) query.UrlParameterProvider
	var defs query.FieldDefinitions

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

		defs = query.NewDefinitions(
			query.Text("name"),
			query.Int("age"),
			query.Float("score"),
			query.Date("birthDate", "2006-01-02"),
			query.JSON("meta").WithSchema(
				query.Text("city"),
			),
		)
	})

	Describe("FromUrlParams", func() {
		It("should handle basic equality", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{"name": "John"})
			urlParams := query.SearchURL(provider, defs).Load(ctx)

			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			build, args := qb.Criteria()

			Expect(build.String()).To(Equal("name = $1"))
			Expect(args).To(Equal([]any{"John"}))
		})

		It("should handle greater than comparison", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{"score_gt": "88.5"})
			urlParams := query.SearchURL(provider, defs).Load(ctx)

			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			build, args := qb.Criteria()

			Expect(build.String()).To(Equal("score > $1"))
			Expect(args).To(Equal([]any{float64(88.5)}))
		})

		It("should handle nested JSON field equality", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{"meta.city": "New York"})
			urlParams := query.SearchURL(provider, defs).Load(ctx)

			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			build, args := qb.Criteria()

			Expect(build.String()).To(Equal("(meta ->> 'city') = $1"))
			Expect(args).To(Equal([]any{"New York"}))
		})

		It("should handle multiple clauses", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{"name": "Jane", "age_lt": "30"})
			urlParams := query.SearchURL(provider, defs).Load(ctx)

			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			build, args := qb.Criteria()

			Expect(build.String()).To(Or(
				Equal("(name = $1) and (age < $2)"),
				Equal("(age < $1) and (name = $2)"),
			))
			Expect(args).To(Or(
				Equal([]any{"Jane", int64(30)}),
				Equal([]any{int64(30), "Jane"}),
			))
		})

		It("should handle numeric between queries", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{"age_bt": "25:35"})
			urlParams := query.SearchURL(provider, defs).Load(ctx)

			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			build, args := qb.Criteria()

			Expect(build.String()).To(Equal("age between $1 and $2"))
			Expect(args).To(Equal([]any{float64(25), float64(35)}))
		})

		It("should handle date between queries", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{"birthDate_bt": "2023-01-01:2023-12-31"})
			urlParams := query.SearchURL(provider, defs).Load(ctx)

			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			build, args := qb.Criteria()

			Expect(build.String()).To(Equal("birth_date between $1 and $2"))
			Expect(len(args)).To(Equal(2))
		})

		It("should handle text like queries", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{"name_lyk": "John%"})
			urlParams := query.SearchURL(provider, defs).Load(ctx)

			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			build, args := qb.Criteria()

			Expect(build.String()).To(Equal("name ilike $1"))
			Expect(args).To(Equal([]any{"John%"}))
		})

		It("should handle in queries", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{"name_in": "John,Jane,Bob"})
			urlParams := query.SearchURL(provider, defs).Load(ctx)

			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			build, args := qb.Criteria()

			Expect(build.String()).To(Equal("name in ($1, $2, $3)"))
			Expect(args).To(Equal([]any{"John", "Jane", "Bob"}))
		})

		It("should handle empty parameters", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{})
			urlParams := query.SearchURL(provider, defs).Load(ctx)

			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			build, args := qb.Criteria()

			Expect(build.String()).To(Equal(""))
			Expect(args).To(BeEmpty())
		})

		It("should handle unknown fields gracefully", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{"unknown_field": "value"})
			urlParams := query.SearchURL(provider, defs).Load(ctx)

			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			build, args := qb.Criteria()

			Expect(build.String()).To(Equal(""))
			Expect(args).To(BeEmpty())
		})
	})

	Describe("SQL Builder Integration", func() {
		It("should work with sort parameters", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"name": "John",
				"sort": "-name,age",
			})

			sortableDefs := query.NewDefinitions(
				query.Text("name").Sortable(),
				query.Int("age").Sortable(),
			)

			urlParams := query.SearchURL(provider, sortableDefs).Load(ctx)

			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			build, args := qb.Criteria()

			Expect(build.String()).To(Equal("name = $1"))
			Expect(args).To(Equal([]any{"John"}))
		})

		It("should work with pagination parameters", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"name":   "John",
				"limit":  "10",
				"offset": "5",
			})

			urlParams := query.SearchURL(provider, defs).Load(ctx)

			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			build, args := qb.Criteria()

			Expect(build.String()).To(Equal("name = $1"))
			Expect(args).To(Equal([]any{"John"}))
		})
	})
})
