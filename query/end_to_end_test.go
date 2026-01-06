package query_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/query"
)

var _ = Describe("End-to-End Query Library Tests", func() {

	var mockParameterProvider func(params map[string]string) query.UrlParameterProvider
	var userFieldDefinitions query.FieldDefinitions
	var transactionFieldDefinitions query.FieldDefinitions

	BeforeEach(func() {
		mockParameterProvider = func(params map[string]string) query.UrlParameterProvider {
			return func(ctx context.Context, name string, _default ...string) contracts.Value {
				if val, ok := params[name]; ok {
					return contracts.Value(val)
				}
				if len(_default) > 0 {
					return contracts.Value(_default[0])
				}
				return ""
			}
		}

		// User field definitions for realistic scenarios
		userFieldDefinitions = query.NewDefinitions(
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
			query.JSON("metadata").WithSchema(
				query.Text("city"),
				query.Int("zip_code"),
			),
		)

		// Transaction field definitions for realistic scenarios
		transactionFieldDefinitions = query.NewDefinitions(
			query.UUID("id"),
			query.Text("reference").Sortable(),
			query.Float("amount").Sortable().WithOperators(
				query.CompareEqual, query.CompareGreaterThan,
				query.CompareLessThan, query.CompareBetween,
			),
			query.Int("account_id").Sortable(),
			query.Text("status").WithOperators(
				query.CompareEqual, query.CompareIn,
			),
			query.Date("transaction_date", "2006-01-02").Sortable(),
			query.JSON("details").WithSchema(
				query.Text("description"),
				query.Text("category"),
			),
		)
	})

	Describe("Complete Flow from URL Parameters to SQL", func() {
		It("should handle complex user search with multiple filters", func() {
			// Simulate URL: /users?username_lyk=john%&age_bt=25:35&active=true&sort_created_at=desc&sort_username=asc&limit=50
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"username_lyk":    "john%",
				"age_bt":          "25:35",
				"active":          "true",
				"sort_created_at": "desc",
				"sort_username":   "asc",
				"limit":           "50",
			})

			// Step 1: Parse URL parameters
			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)

			// Step 2: Verify parsed parameters
			Expect(urlParams.GetConditions()).To(HaveLen(3)) // username, age between, active
			Expect(urlParams.GetSorts()).To(HaveLen(2))      // created_at desc, username asc
			Expect(urlParams.GetLimit()).To(Equal(int64(50)))

			// Step 3: Build SQL query
			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			sqlQuery, args := qb.Criteria()

			// Step 4: Verify SQL generation
			Expect(sqlQuery.String()).To(ContainSubstring("username ilike"))
			Expect(sqlQuery.String()).To(ContainSubstring("age between"))
			Expect(sqlQuery.String()).To(ContainSubstring("active ="))
			Expect(len(args)).To(Equal(4)) // username, age_from, age_to, active
		})

		It("should handle transaction search with date range and sorting", func() {
			// Simulate URL: /transactions?amount_gt=100&account_id=123&status_in=completed,pending&transaction_date_bt=2023-01-01:2023-12-31&sort_amount=desc
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"amount_gt":           "100",
				"account_id":          "123",
				"status_in":           "completed,pending",
				"transaction_date_bt": "2023-01-01:2023-12-31",
				"sort_amount":         "desc",
			})

			// Step 1: Parse URL parameters
			urlParams := query.SearchURL(provider, transactionFieldDefinitions).Load(ctx)

			// Step 2: Verify parsed parameters
			conditions := urlParams.GetConditions()
			Expect(conditions).To(HaveLen(4))                 // amount, account_id, status_in, date_between
			Expect(urlParams.GetSorts()).To(HaveLen(1))       // amount desc
			Expect(urlParams.GetLimit()).To(Equal(int64(20))) // default limit

			// Step 3: Build SQL query
			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			sqlQuery, args := qb.Criteria()

			// Step 4: Verify SQL generation
			Expect(sqlQuery.String()).To(ContainSubstring("amount >"))
			Expect(sqlQuery.String()).To(ContainSubstring("account_id ="))
			Expect(sqlQuery.String()).To(ContainSubstring("status in"))
			Expect(sqlQuery.String()).To(ContainSubstring("transaction_date between"))
			Expect(len(args)).To(Equal(6)) // amount, account_id, status_1, status_2, date_from, date_to
		})

		It("should handle JSON field queries", func() {
			// Simulate URL: /users?metadata.city=New York&metadata.zip_code_gt=10000
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"metadata.city":        "New York",
				"metadata.zip_code_gt": "10000",
			})

			// Step 1: Parse URL parameters
			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)

			// Step 2: Verify parsed parameters
			conditions := urlParams.GetConditions()
			Expect(conditions).To(HaveLen(2))

			// Step 3: Build SQL query
			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			sqlQuery, args := qb.Criteria()

			// Step 4: Verify SQL generation
			Expect(sqlQuery.String()).To(ContainSubstring("(metadata ->> 'city') ="))
			Expect(sqlQuery.String()).To(ContainSubstring("(metadata ->> 'zip_code') >"))
			Expect(len(args)).To(Equal(2))
		})

		It("should handle pagination with page and limit", func() {
			// Simulate URL: /users?active=true&page=2&limit=10
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"active": "true",
				"page":   "2",
				"limit":  "10",
			})

			// Step 1: Parse URL parameters
			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)

			// Step 2: Verify parsed parameters
			Expect(urlParams.GetLimit()).To(Equal(int64(10)))
			Expect(urlParams.GetOffset()).To(Equal(int64(10))) // (page-1)*limit = 10

			// Step 3: Build SQL query
			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			sqlQuery, args := qb.Criteria()

			// Step 4: Verify SQL generation
			Expect(sqlQuery.String()).To(ContainSubstring("active ="))
			Expect(len(args)).To(Equal(1))
		})
	})

	Describe("API Integration Scenarios", func() {
		It("should work with api.ListRequest integration", func() {
			// Simulate api.ListRequest with query parameters
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"username_lyk": "admin%",
				"age_gt":       "18",
				"sort":         "username",
				"limit":        "25",
			})

			// This simulates how api.ListRequest would work
			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)

			// Build the complete query (as would happen in a service)
			qb := query.SQLBuilder[any](nil, nil).
				FromUrlParams(urlParams).
				Count() // Enable count query

			// Get the SQL preview (as would be used for logging/debugging)
			sqlPreview, args := qb.SelectQueryPreview("users")

			Expect(sqlPreview).To(ContainSubstring("select * from users"))
			Expect(sqlPreview).To(ContainSubstring("username ilike"))
			Expect(sqlPreview).To(ContainSubstring("age >"))
			Expect(sqlPreview).To(ContainSubstring("order by username"))
			Expect(sqlPreview).To(ContainSubstring("limit 25"))
			Expect(len(args)).To(Equal(2))
		})

		It("should handle complex boolean logic with multiple conditions", func() {
			// Simulate URL: /users?username_lyk=john%&age_bt=25:35&active=true&email_in=john@example.com,jane@example.com
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"username_lyk": "john%",
				"age_bt":       "25:35",
				"active":       "true",
				"email_in":     "john@example.com,jane@example.com",
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			sqlQuery, args := qb.Criteria()

			// All conditions should be combined with AND
			Expect(sqlQuery.String()).To(ContainSubstring("username ilike"))
			Expect(sqlQuery.String()).To(ContainSubstring("age between"))
			Expect(sqlQuery.String()).To(ContainSubstring("active ="))
			Expect(sqlQuery.String()).To(ContainSubstring("email in"))
			Expect(sqlQuery.String()).To(ContainSubstring("and"))
			Expect(len(args)).To(Equal(6)) // username, age_from, age_to, active, email_1, email_2
		})
	})

	Describe("Error Handling and Edge Cases", func() {
		It("should handle invalid parameter formats gracefully", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"age_bt":       "invalid:range", // Invalid between format
				"username_lyk": "john%",
				"active":       "not_boolean", // Invalid boolean
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			sqlQuery, args := qb.Criteria()

			// Should only include valid conditions
			Expect(sqlQuery.String()).To(ContainSubstring("username ilike"))
			Expect(sqlQuery.String()).ToNot(ContainSubstring("age between"))
			Expect(sqlQuery.String()).ToNot(ContainSubstring("active ="))
			Expect(len(args)).To(Equal(1))
		})

		It("should handle empty query parameters", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			sqlQuery, args := qb.Criteria()

			Expect(sqlQuery.String()).To(BeEmpty())
			Expect(args).To(BeEmpty())
		})

		It("should handle unknown parameters gracefully", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"unknown_field": "value",
				"username":      "john",
				"unknown_param": "another_value",
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			sqlQuery, args := qb.Criteria()

			// Should only include known fields
			Expect(sqlQuery.String()).To(ContainSubstring("username ="))
			Expect(sqlQuery.String()).ToNot(ContainSubstring("unknown_field"))
			Expect(len(args)).To(Equal(1))
		})
	})

	Describe("Realistic Production Scenarios", func() {
		It("should handle dashboard-style transaction filtering", func() {
			// Simulate a dashboard request with multiple filters
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"amount_gt":           "1000",
				"status_in":           "completed,pending",
				"transaction_date_bt": "2023-01-01:2023-12-31",
				"account_id":          "12345",
				"sort":                "-transaction_date,amount",
				"limit":               "100",
			})

			urlParams := query.SearchURL(provider, transactionFieldDefinitions).Load(ctx)
			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			sqlQuery, args := qb.Criteria()

			Expect(sqlQuery.String()).To(ContainSubstring("amount >"))
			Expect(sqlQuery.String()).To(ContainSubstring("status in"))
			Expect(sqlQuery.String()).To(ContainSubstring("transaction_date between"))
			Expect(sqlQuery.String()).To(ContainSubstring("account_id ="))
			Expect(len(args)).To(Equal(6)) // amount, status_1, status_2, date_from, date_to, account_id
		})

		It("should handle user search with complex text matching", func() {
			// Simulate user search with various text matching patterns
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"username_lyk":  "john%",
				"email":         "john@example.com",
				"metadata.city": "New York",
				"active":        "true",
				"sort":          "username",
			})

			urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			sqlQuery, args := qb.Criteria()

			Expect(sqlQuery.String()).To(ContainSubstring("username ilike"))
			Expect(sqlQuery.String()).To(ContainSubstring("email ="))
			Expect(sqlQuery.String()).To(ContainSubstring("(metadata ->> 'city') ="))
			Expect(sqlQuery.String()).To(ContainSubstring("active ="))
			Expect(len(args)).To(Equal(4))
		})
	})
})
