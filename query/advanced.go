package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// JoinType represents different types of SQL joins
type JoinType string

const (
	InnerJoin JoinType = "INNER JOIN"
	LeftJoin  JoinType = "LEFT JOIN"
	RightJoin JoinType = "RIGHT JOIN"
	FullJoin  JoinType = "FULL OUTER JOIN"
	CrossJoin JoinType = "CROSS JOIN"
)

// Join represents a SQL join clause
type Join struct {
	Type      JoinType
	Table     string
	Alias     string
	Condition string
}

// Subquery represents a SQL subquery
type Subquery struct {
	Query string
	Alias string
	Args  []interface{}
}

// AggregateFunc represents an aggregate function
type AggregateFunc string

const (
	Count   AggregateFunc = "COUNT"
	Sum     AggregateFunc = "SUM"
	Avg     AggregateFunc = "AVG"
	Min     AggregateFunc = "MIN"
	Max     AggregateFunc = "MAX"
	GroupConcat AggregateFunc = "STRING_AGG"
)

// Aggregate represents an aggregate operation
type Aggregate struct {
	Function AggregateFunc
	Column   string
	Alias    string
	Distinct bool
}

// AdvancedQueryBuilder provides advanced query building capabilities
type AdvancedQueryBuilder struct {
	baseTable   string
	tableAlias  string
	columns     []string
	joins       []Join
	where       []string
	whereArgs   []interface{}
	groupBy     []string
	having      []string
	havingArgs  []interface{}
	orderBy     []string
	limit       int
	offset      int
	aggregates  []Aggregate
	subqueries  []Subquery
	unions      []string
	cte         map[string]string // Common Table Expressions
}

// NewAdvancedQueryBuilder creates a new advanced query builder
func NewAdvancedQueryBuilder(table string, alias ...string) *AdvancedQueryBuilder {
	qb := &AdvancedQueryBuilder{
		baseTable:  table,
		columns:    []string{},
		joins:      []Join{},
		where:      []string{},
		whereArgs:  []interface{}{},
		groupBy:    []string{},
		having:     []string{},
		havingArgs: []interface{}{},
		orderBy:    []string{},
		aggregates: []Aggregate{},
		subqueries: []Subquery{},
		unions:     []string{},
		cte:        make(map[string]string),
	}

	if len(alias) > 0 {
		qb.tableAlias = alias[0]
	}

	return qb
}

// Select specifies which columns to select
func (qb *AdvancedQueryBuilder) Select(columns ...string) *AdvancedQueryBuilder {
	qb.columns = append(qb.columns, columns...)
	return qb
}

// SelectAggregate adds an aggregate function
func (qb *AdvancedQueryBuilder) SelectAggregate(agg Aggregate) *AdvancedQueryBuilder {
	qb.aggregates = append(qb.aggregates, agg)
	return qb
}

// Join adds a join clause
func (qb *AdvancedQueryBuilder) Join(join Join) *AdvancedQueryBuilder {
	qb.joins = append(qb.joins, join)
	return qb
}

// InnerJoin adds an inner join
func (qb *AdvancedQueryBuilder) InnerJoin(table, alias, condition string) *AdvancedQueryBuilder {
	return qb.Join(Join{
		Type:      InnerJoin,
		Table:     table,
		Alias:     alias,
		Condition: condition,
	})
}

// LeftJoin adds a left join
func (qb *AdvancedQueryBuilder) LeftJoin(table, alias, condition string) *AdvancedQueryBuilder {
	return qb.Join(Join{
		Type:      LeftJoin,
		Table:     table,
		Alias:     alias,
		Condition: condition,
	})
}

// RightJoin adds a right join
func (qb *AdvancedQueryBuilder) RightJoin(table, alias, condition string) *AdvancedQueryBuilder {
	return qb.Join(Join{
		Type:      RightJoin,
		Table:     table,
		Alias:     alias,
		Condition: condition,
	})
}

// Where adds a WHERE condition
func (qb *AdvancedQueryBuilder) Where(condition string, args ...interface{}) *AdvancedQueryBuilder {
	qb.where = append(qb.where, condition)
	qb.whereArgs = append(qb.whereArgs, args...)
	return qb
}

// WhereIn adds a WHERE IN condition
func (qb *AdvancedQueryBuilder) WhereIn(column string, values []interface{}) *AdvancedQueryBuilder {
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = fmt.Sprintf("$%d", len(qb.whereArgs)+i+1)
	}
	condition := fmt.Sprintf("%s IN (%s)", column, strings.Join(placeholders, ", "))
	qb.where = append(qb.where, condition)
	qb.whereArgs = append(qb.whereArgs, values...)
	return qb
}

// WhereExists adds a WHERE EXISTS subquery
func (qb *AdvancedQueryBuilder) WhereExists(subquery string, args ...interface{}) *AdvancedQueryBuilder {
	condition := fmt.Sprintf("EXISTS (%s)", subquery)
	qb.where = append(qb.where, condition)
	qb.whereArgs = append(qb.whereArgs, args...)
	return qb
}

// GroupBy adds GROUP BY columns
func (qb *AdvancedQueryBuilder) GroupBy(columns ...string) *AdvancedQueryBuilder {
	qb.groupBy = append(qb.groupBy, columns...)
	return qb
}

// Having adds a HAVING condition
func (qb *AdvancedQueryBuilder) Having(condition string, args ...interface{}) *AdvancedQueryBuilder {
	qb.having = append(qb.having, condition)
	qb.havingArgs = append(qb.havingArgs, args...)
	return qb
}

// OrderBy adds ORDER BY clauses
func (qb *AdvancedQueryBuilder) OrderBy(columns ...string) *AdvancedQueryBuilder {
	qb.orderBy = append(qb.orderBy, columns...)
	return qb
}

// Limit sets the LIMIT
func (qb *AdvancedQueryBuilder) Limit(limit int) *AdvancedQueryBuilder {
	qb.limit = limit
	return qb
}

// Offset sets the OFFSET
func (qb *AdvancedQueryBuilder) Offset(offset int) *AdvancedQueryBuilder {
	qb.offset = offset
	return qb
}

// WithSubquery adds a subquery to the FROM clause
func (qb *AdvancedQueryBuilder) WithSubquery(subquery Subquery) *AdvancedQueryBuilder {
	qb.subqueries = append(qb.subqueries, subquery)
	return qb
}

// Union adds a UNION query
func (qb *AdvancedQueryBuilder) Union(query string) *AdvancedQueryBuilder {
	qb.unions = append(qb.unions, query)
	return qb
}

// WithCTE adds a Common Table Expression
func (qb *AdvancedQueryBuilder) WithCTE(name, query string) *AdvancedQueryBuilder {
	qb.cte[name] = query
	return qb
}

// Build constructs the final SQL query
func (qb *AdvancedQueryBuilder) Build() (string, []interface{}) {
	var queryParts []string
	args := make([]interface{}, 0)

	// Add CTEs
	if len(qb.cte) > 0 {
		ctes := make([]string, 0, len(qb.cte))
		for name, query := range qb.cte {
			ctes = append(ctes, fmt.Sprintf("%s AS (%s)", name, query))
		}
		queryParts = append(queryParts, "WITH "+strings.Join(ctes, ", "))
	}

	// SELECT clause
	selectClause := "SELECT "
	if len(qb.columns) == 0 && len(qb.aggregates) == 0 {
		selectClause += "*"
	} else {
		var selections []string

		// Add regular columns
		selections = append(selections, qb.columns...)

		// Add aggregates
		for _, agg := range qb.aggregates {
			aggStr := string(agg.Function) + "("
			if agg.Distinct {
				aggStr += "DISTINCT "
			}
			aggStr += agg.Column + ")"
			if agg.Alias != "" {
				aggStr += " AS " + agg.Alias
			}
			selections = append(selections, aggStr)
		}

		selectClause += strings.Join(selections, ", ")
	}
	queryParts = append(queryParts, selectClause)

	// FROM clause
	fromClause := "FROM " + qb.baseTable
	if qb.tableAlias != "" {
		fromClause += " AS " + qb.tableAlias
	}
	queryParts = append(queryParts, fromClause)

	// Add subqueries to FROM
	for _, sq := range qb.subqueries {
		fromClause += fmt.Sprintf(", (%s) AS %s", sq.Query, sq.Alias)
		args = append(args, sq.Args...)
	}

	// JOIN clauses
	for _, join := range qb.joins {
		joinClause := string(join.Type) + " " + join.Table
		if join.Alias != "" {
			joinClause += " AS " + join.Alias
		}
		joinClause += " ON " + join.Condition
		queryParts = append(queryParts, joinClause)
	}

	// WHERE clause
	if len(qb.where) > 0 {
		queryParts = append(queryParts, "WHERE "+strings.Join(qb.where, " AND "))
		args = append(args, qb.whereArgs...)
	}

	// GROUP BY clause
	if len(qb.groupBy) > 0 {
		queryParts = append(queryParts, "GROUP BY "+strings.Join(qb.groupBy, ", "))
	}

	// HAVING clause
	if len(qb.having) > 0 {
		queryParts = append(queryParts, "HAVING "+strings.Join(qb.having, " AND "))
		args = append(args, qb.havingArgs...)
	}

	// ORDER BY clause
	if len(qb.orderBy) > 0 {
		queryParts = append(queryParts, "ORDER BY "+strings.Join(qb.orderBy, ", "))
	}

	// LIMIT clause
	if qb.limit > 0 {
		queryParts = append(queryParts, fmt.Sprintf("LIMIT %d", qb.limit))
	}

	// OFFSET clause
	if qb.offset > 0 {
		queryParts = append(queryParts, fmt.Sprintf("OFFSET %d", qb.offset))
	}

	query := strings.Join(queryParts, " ")

	// Add UNION queries
	if len(qb.unions) > 0 {
		for _, union := range qb.unions {
			query += " UNION " + union
		}
	}

	return query, args
}

// Execute executes the query
func (qb *AdvancedQueryBuilder) Execute(ctx context.Context, db *pgxpool.Pool) ([]map[string]interface{}, error) {
	query, args := qb.Build()
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		row := make(map[string]interface{})
		for i, col := range rows.FieldDescriptions() {
			row[string(col.Name)] = values[i]
		}
		results = append(results, row)
	}

	return results, nil
}

// Count returns the count of rows
func (qb *AdvancedQueryBuilder) Count(ctx context.Context, db *pgxpool.Pool) (int64, error) {
	// Create a new query builder for the count query
	countQB := &AdvancedQueryBuilder{
		baseTable:  qb.baseTable,
		tableAlias: qb.tableAlias,
		joins:      qb.joins,
		where:      qb.where,
		whereArgs:  qb.whereArgs,
	}

	countQB.SelectAggregate(Aggregate{
		Function: Count,
		Column:   "*",
		Alias:    "count",
	})

	query, args := countQB.Build()
	var count int64
	err := db.QueryRow(ctx, query, args...).Scan(&count)
	return count, err
}

// ExampleUsage demonstrates how to use the advanced query builder
func ExampleUsage(ctx context.Context, db *pgxpool.Pool) {
	// Example 1: Complex join with aggregation
	qb1 := NewAdvancedQueryBuilder("orders", "o").
		Select("o.order_id", "o.customer_id", "c.customer_name").
		SelectAggregate(Aggregate{Function: Sum, Column: "oi.amount", Alias: "total_amount"}).
		SelectAggregate(Aggregate{Function: Count, Column: "oi.item_id", Alias: "item_count"}).
		InnerJoin("customers", "c", "o.customer_id = c.customer_id").
		InnerJoin("order_items", "oi", "o.order_id = oi.order_id").
		Where("o.order_date > $1", "2024-01-01").
		GroupBy("o.order_id", "o.customer_id", "c.customer_name").
		Having("SUM(oi.amount) > $1", 1000).
		OrderBy("total_amount DESC").
		Limit(10)

	results1, _ := qb1.Execute(ctx, db)
	fmt.Printf("Complex join results: %v\n", results1)

	// Example 2: Subquery in WHERE clause
	qb2 := NewAdvancedQueryBuilder("products", "p").
		Select("p.product_id", "p.product_name", "p.price").
		WhereExists("SELECT 1 FROM order_items oi WHERE oi.product_id = p.product_id").
		OrderBy("p.price DESC")

	results2, _ := qb2.Execute(ctx, db)
	fmt.Printf("Subquery results: %v\n", results2)

	// Example 3: CTE (Common Table Expression)
	qb3 := NewAdvancedQueryBuilder("customer_orders", "co").
		WithCTE("customer_orders", `
			SELECT customer_id, COUNT(*) as order_count, SUM(total) as total_spent
			FROM orders
			GROUP BY customer_id
		`).
		Select("co.customer_id", "co.order_count", "co.total_spent", "c.customer_name").
		InnerJoin("customers", "c", "co.customer_id = c.customer_id").
		Where("co.total_spent > $1", 5000).
		OrderBy("co.total_spent DESC")

	results3, _ := qb3.Execute(ctx, db)
	fmt.Printf("CTE results: %v\n", results3)
}
