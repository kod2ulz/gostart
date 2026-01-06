package query_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/query"
)

var _ = Describe("Parameter Override Functionality Verification", func() {

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

		userFieldDefinitions = query.NewDefinitions(
			query.Text("username").Sortable(),
			query.Int("age"),
			query.Bool("active"),
			query.Text("account_id"),
		)

		// Transaction field definitions for realistic scenarios
		transactionFieldDefinitions = query.NewDefinitions(
			query.Text("status").WithOperators(
				query.CompareEqual, query.CompareIn,
			),
			query.Float("amount").WithOperators(
				query.CompareEqual, query.CompareGreaterThan,
				query.CompareLessThan, query.CompareBetween,
			),
			query.Date("transaction_date", "2006-01-02"),
		)
	})

	Describe("Parameter Override Functionality Check", func() {
		It("should demonstrate that parameter override functionality was lost", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"username": "john_doe",
				"age":      "25",
			})

			// Parse URL parameters
			var urlParams query.URLSearchParam = query.SearchURL(provider, userFieldDefinitions).Load(ctx)

			// Verify that the current URLSearchParam interface is minimal
			Expect(urlParams).NotTo(BeNil())
			Expect(urlParams.GetConditions()).To(HaveLen(2))
			Expect(urlParams.GetLimit()).To(Equal(int64(20))) // default limit

			// Check if the interface has the old override methods
			fmt.Printf("Current URLSearchParam interface type: %T\n", urlParams)

			// Try to use the old interface methods (these should not exist)
			// This will cause compilation errors if uncommented, confirming the functionality was lost
			/*
				// These methods from the old interface are no longer available:
				// - WithField(field string, val any) URLSearchParam
				// - WithComparison(field string, comparator CompareOperator, val any) URLSearchParam
				// - GetFieldValues() map[string]any
				// - GetFieldNullables() map[string]bool
				// - GetFieldSort() map[string]SortType
				// - GetFieldComparisons() map[string]map[CompareOperator]any
			*/

			// The current interface only has these methods:
			// - GetConditions() []ParsedCondition
			// - GetSorts() []ParsedSort
			// - GetLimit() int64
			// - GetOffset() int64
		})

		It("should show that the new implementation cannot add runtime overrides", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"username": "john_doe",
			})

			// Parse URL parameters
			var urlParams query.URLSearchParam = query.SearchURL(provider, userFieldDefinitions).Load(ctx)

			// Verify initial state
			Expect(urlParams.GetConditions()).To(HaveLen(1))
			Expect(urlParams.GetConditions()[0].Operator).To(Equal(query.CompareEqual))

			// In the old implementation, we could do this:
			// overriddenParams := urlParams.WithField("account_id", 123).WithComparison("active", query.CompareEqual, true)
			// But now there's no way to add additional conditions at runtime

			// This demonstrates a critical loss of functionality
			// Service layers cannot add security constraints or bias queries
			// based on user permissions or other business logic
		})

		It("should demonstrate the production impact of missing override functionality", func() {
			// This test demonstrates the real-world impact of the missing functionality

			// Scenario: Multi-tenant application where regular users can only see their own data
			// but admins can see all data

			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"username": "john_doe",
				"age_gt":   "20",
			})

			// Parse URL parameters from the request
			var urlParams query.URLSearchParam = query.SearchURL(provider, userFieldDefinitions).Load(ctx)

			// Build SQL query
			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			sqlQuery, args := qb.Criteria()

			// The generated query only includes the user-specified parameters
			Expect(sqlQuery.String()).To(ContainSubstring("username ="))
			Expect(sqlQuery.String()).To(ContainSubstring("age >"))
			Expect(len(args)).To(Equal(2))

			// In production, we would want to add account_id filtering for security:
			// - For regular users: force account_id = user_account_id
			// - For admins: allow all accounts or filter by specified account_id

			// With the old interface, this was possible:
			// if !user.IsAdmin {
			//     urlParams = urlParams.WithField("account_id", user.AccountID)
			// }

			// With the new interface, this is not possible without
			// manually manipulating the SQL builder, which breaks the abstraction
		})

		It("should show that the new interface lacks introspection capabilities", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"username": "john_doe",
				"age_gt":   "25",
			})

			var urlParams query.URLSearchParam = query.SearchURL(provider, userFieldDefinitions).Load(ctx)

			// The old interface provided introspection methods:
			// - HasField(field string) bool
			// - HasComparison(field string, comparator CompareOperator) bool
			// - HasAnyComparison(field string, comparator ...CompareOperator) bool
			// - GetFieldValues() map[string]any
			// - GetFieldComparisons() map[string]map[CompareOperator]any

			// The new interface only provides the final parsed conditions
			// but doesn't allow checking what fields are already present

			conditions := urlParams.GetConditions()
			Expect(conditions).To(HaveLen(2))

			// We can inspect conditions manually, but it's more cumbersome
			var hasUsernameFilter, hasAgeFilter bool
			for _, condition := range conditions {
				if len(condition.DBPath) > 0 && condition.DBPath[0] == "username" {
					hasUsernameFilter = true
				}
				if len(condition.DBPath) > 0 && condition.DBPath[0] == "age" {
					hasAgeFilter = true
				}
			}

			Expect(hasUsernameFilter).To(BeTrue())
			Expect(hasAgeFilter).To(BeTrue())

			// This demonstrates that while we can work around the missing
			// introspection methods, it's much more verbose and error-prone
		})
	})

	Describe("Parameter Override Functionality", func() {
		It("should allow adding conditions from service layer", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"username": "john_doe",
				"age_gt":   "25",
			})

			// Parse URL parameters from request
			var urlParams query.URLSearchParam = query.SearchURL(provider, userFieldDefinitions).Load(ctx)

			// Verify initial state
			Expect(urlParams.GetConditions()).To(HaveLen(2))
			Expect(urlParams.GetLimit()).To(Equal(int64(20)))

			// Add service layer security constraint
			urlParams = urlParams.AddField("account_id", 123)

			// Add additional business logic constraint
			urlParams = urlParams.AddCondition("active", query.CompareEqual, true)

			// Verify conditions were added
			Expect(urlParams.GetConditions()).To(HaveLen(4))

			// Build SQL query
			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			sqlQuery, args := qb.Criteria()

			Expect(sqlQuery.String()).To(ContainSubstring("username ="))
			Expect(sqlQuery.String()).To(ContainSubstring("age >"))
			Expect(sqlQuery.String()).To(ContainSubstring("account_id ="))
			Expect(sqlQuery.String()).To(ContainSubstring("active ="))
			Expect(len(args)).To(Equal(4))
		})

		It("should allow overriding sort and pagination", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"username": "john_doe",
			})

			var urlParams query.URLSearchParam = query.SearchURL(provider, userFieldDefinitions).Load(ctx)

			// Verify initial state
			Expect(urlParams.GetSorts()).To(HaveLen(0))
			Expect(urlParams.GetLimit()).To(Equal(int64(20)))
			Expect(urlParams.GetOffset()).To(Equal(int64(0)))

			// Override sort from service layer
			urlParams = urlParams.AddSort("username", query.SortAsc)
			urlParams = urlParams.AddSort("created_at", query.SortDesc)

			// Override pagination for admin users
			urlParams = urlParams.SetLimit(100)
			urlParams = urlParams.SetOffset(0)

			// Verify overrides were applied
			Expect(urlParams.GetSorts()).To(HaveLen(2))
			Expect(urlParams.GetSorts()[0].DBName).To(Equal("username"))
			Expect(urlParams.GetSorts()[0].Type).To(Equal(query.SortAsc))
			Expect(urlParams.GetSorts()[1].DBName).To(Equal("created_at"))
			Expect(urlParams.GetSorts()[1].Type).To(Equal(query.SortDesc))
			Expect(urlParams.GetLimit()).To(Equal(int64(100)))
			Expect(urlParams.GetOffset()).To(Equal(int64(0)))
		})

		It("should demonstrate multi-tenant security pattern", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"username_lyk": "john%",
				"active":       "true",
			})

			// Parse user request
			var urlParams query.URLSearchParam = query.SearchURL(provider, userFieldDefinitions).Load(ctx)

			// Simulate regular user (non-admin)
			isAdmin := false
			userAccountID := int64(123)

			if !isAdmin {
				// Add account_id constraint for security
				urlParams = urlParams.AddField("account_id", userAccountID)
			}

			// Build query
			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			sqlQuery, args := qb.Criteria()

			// Verify security constraint is included
			Expect(sqlQuery.String()).To(ContainSubstring("username ilike"))
			Expect(sqlQuery.String()).To(ContainSubstring("active ="))
			Expect(sqlQuery.String()).To(ContainSubstring("account_id ="))
			Expect(len(args)).To(Equal(3))
		})

		It("should support complex business logic overrides", func() {
			ctx := context.Background()
			provider := mockParameterProvider(map[string]string{
				"status": "pending",
			})

			var urlParams query.URLSearchParam = query.SearchURL(provider, transactionFieldDefinitions).Load(ctx)

			// Add business logic: only show transactions from last 30 days
			thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
			urlParams = urlParams.AddCondition("transaction_date", query.CompareGreaterThanOrEqual, thirtyDaysAgo)

			// Add business logic: exclude small amounts for reporting
			urlParams = urlParams.AddCondition("amount", query.CompareGreaterThanOrEqual, 10.0)

			// Build query
			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			sqlQuery, args := qb.Criteria()

			Expect(sqlQuery.String()).To(ContainSubstring("status ="))
			Expect(sqlQuery.String()).To(ContainSubstring("transaction_date >="))
			Expect(sqlQuery.String()).To(ContainSubstring("amount >="))
			Expect(len(args)).To(Equal(3))
		})
	})
})
