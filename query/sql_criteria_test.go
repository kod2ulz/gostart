package query_test

import (
	"fmt"
	"testing"

	"github.com/kod2ulz/gostart/query"
	"github.com/stretchr/testify/assert"
)

func arg(name string, num int, op ...string) string {
	if len(op) == 0 {
		op = []string{"="}
	}
	switch op[0] {
	case query.SELECT_LIKE:
		return fmt.Sprintf("%s %s $%d", name, op[0], num)
	default:
		return fmt.Sprintf("%s%s$%d", name, op[0], num)
	}
}

var criteria_test_cases = []struct {
	conditions []query.Condition
	args       []interface{}
	expected   string
}{
	{
		conditions: []query.Condition{query.Equal("name", "someone")},
		expected:   arg("name", 1), args: []interface{}{"someone"},
	},
	{
		conditions: []query.Condition{query.Or(query.Equal("fname", "john"), query.Equal("lname", "doe"))},
		expected:   fmt.Sprintf("(%s) %s (%s)", arg("fname", 1), query.WhereOr, arg("lname", 2)),
		args:       []interface{}{"john", "doe"},
	},
	{
		conditions: []query.Condition{query.Or(query.GreaterThan("age", 23), query.Like("lname", "doe"))},
		expected:   fmt.Sprintf("(%s) %s (%s)", arg("age", 1, ">"), query.WhereOr, arg("lname", 2, query.SELECT_LIKE)),
		args:       []interface{}{23, "doe"},
	},
	{
		conditions: []query.Condition{query.Or(query.GreaterThanOrEqual("age", 23), query.Like("lname", "doe"))},
		expected:   fmt.Sprintf("(%s) %s (%s)", arg("age", 1, ">="), query.WhereOr, arg("lname", 2, query.SELECT_LIKE)),
		args:       []interface{}{23, "doe"},
	},
	{
		conditions: []query.Condition{query.Or(query.LessThan("age", 23), query.Like("lname", "doe"))},
		expected:   fmt.Sprintf("(%s) %s (%s)", arg("age", 1, "<"), query.WhereOr, arg("lname", 2, query.SELECT_LIKE)),
		args:       []interface{}{23, "doe"},
	},
	{
		conditions: []query.Condition{query.Or(query.LessThanOrEqual("age", 23), query.Like("lname", "doe"))},
		expected:   fmt.Sprintf("(%s) %s (%s)", arg("age", 1, "<="), query.WhereOr, arg("lname", 2, query.SELECT_LIKE)),
		args:       []interface{}{23, "doe"},
	},
	{
		conditions: []query.Condition{query.Or(query.Null("age", "dob", "country")), query.Like("lname", "doe")},
		expected:   fmt.Sprintf("((%s is null) or (%s is null) or (%s is null)) %s (%s)", "age", "dob", "country", query.WhereAnd, arg("lname", 1, query.SELECT_LIKE)),
		args:       []interface{}{"doe"},
	},
	{
		conditions: []query.Condition{query.Or(query.NotNull("age", "dob", "country")), query.Like("lname", "doe")},
		expected:   fmt.Sprintf("((%s is not null) or (%s is not null) or (%s is not null)) %s (%s)", "age", "dob", "country", query.WhereAnd, arg("lname", 1, query.SELECT_LIKE)),
		args:       []interface{}{"doe"},
	},
}

func TestWhereConditionBuilder(t *testing.T) {
	for _, tc := range criteria_test_cases {
		qb := query.SQLBuilder[any](nil, nil)
		build, args := qb.Where(tc.conditions...).Criteria()
		assert.Equal(t, tc.expected, build.String())
		assert.Equal(t, tc.args, args)
	}
}
