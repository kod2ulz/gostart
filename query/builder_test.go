package query_test

import (
	"context"
	"testing"

	"github.com/kod2ulz/gostart/config"
	"github.com/kod2ulz/gostart/query"
	"github.com/stretchr/testify/assert"
)

// mockParameterProvider simulates reading from a URL query map for testing.
func mockParameterProvider(params map[string]string) query.UrlParameterProvider {
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

func TestSQLBuilderFromUrl(t *testing.T) {
	defs := query.NewDefinitions(
		query.Text("name"),
		query.Int("age"),
		query.Float("score"),
		query.Date("birthDate", "2006-01-02"),
		query.JSON("meta").WithSchema(
			query.Text("city"),
		),
	)

	testCases := []struct {
		name         string
		params       map[string]string
		expectedSQL  string
		expectedArgs []interface{}
	}{
		{
			name:         "Basic Equality",
			params:       map[string]string{"name": "John"},
			expectedSQL:  "name = $1",
			expectedArgs: []interface{}{"John"},
		},
		{
			name:         "Greater Than",
			params:       map[string]string{"score_gt": "88.5"},
			expectedSQL:  "score > $1",
			expectedArgs: []interface{}{float64(88.5)},
		},
		{
			name:         "Nested JSON Field Equality",
			params:       map[string]string{"meta.city": "New York"},
			expectedSQL:  "(meta ->> 'city') = $1",
			expectedArgs: []interface{}{"New York"},
		},
		{
			name: "Multiple Clauses",
			params: map[string]string{"name": "Jane", "age_lt": "30"},
			expectedSQL:  "(name = $1) and (age < $2)",
			expectedArgs: []interface{}{"Jane", int64(30)},
		},
	{
			name: "Numeric Between",
			params: map[string]string{"age_bt": "25:35"},
			expectedSQL:  "age between $1 and $2",
			expectedArgs: []interface{}{float64(25), float64(35)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			provider := mockParameterProvider(tc.params)
			urlParams := query.SearchURL(provider, defs).Load(ctx)

			qb := query.SQLBuilder[any](nil, nil).FromUrlParams(urlParams)
			build, args := qb.Criteria()

			// The ElementsMatch assertion is tricky with argument order ($1, $2).
			// For this test, we will rely on the fact that our parser processes fields in a stable order.
			// A more advanced test could parse the SQL, but this is sufficient for now.
			assert.Equal(t, tc.expectedSQL, build.String())
			assert.Equal(t, tc.expectedArgs, args)
		})
	}
}