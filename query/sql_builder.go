package query

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/kod2ulz/gostart/sqlc"
	"github.com/kod2ulz/gostart/utils"
)

var env = utils.Env.Helper("SQL_QUERY_BUILDER")

var (
	SELECT_COUNT_FIELDS = env.Get("SELECT_COUNT_FIELDS", "*").StringList()
	SELECT_FIELDS       = env.Get("SELECT_FIELDS", "*").StringList()
	SELECT_LIKE         = env.Get("SELECT_LIKE", "ilike").String()
	SELECT_LIMIT        = env.Get("SELECT_LIMIT", "20").Int64()
	DEFAULT_SCHEMA      = env.Get("DEFAULT_SCHEMA", "public").String()
)

type RowScanFunc[T any] func(pgx.Rows) (T, error)

func SQLBuilder[T any](dbtx sqlc.DBTX, rowScanner RowScanFunc[T]) *sqlBuilder[T] {
	return &sqlBuilder[T]{
		selectFields: SELECT_FIELDS,
		countFields:  SELECT_COUNT_FIELDS,
		orderBy:      []string{},
		limit:        SELECT_LIMIT,
		offset:       0,
		count:        false,
		rowScanner:   rowScanner,
		dbtx:         dbtx,
	}
}

type sqlBuilder[T any] struct {
	selectFields []string
	countFields  []string
	orderBy      []string
	limit        int64
	offset       int64
	count        bool
	rowScanner   RowScanFunc[T]
	where        *WhereCriteria
	dbtx         sqlc.DBTX
}

func (sb *sqlBuilder[T]) Count(fields ...string) *sqlBuilder[T] {
	if len(fields) > 0 {
		sb.countFields = fields
	}
	sb.count = true
	return sb
}

func (sb *sqlBuilder[T]) Limit(limit int64) *sqlBuilder[T] {
	if limit > 0 {
		sb.limit = limit
	}
	return sb
}

func (sb *sqlBuilder[T]) Offset(offset int64) *sqlBuilder[T] {
	if offset > 0 {
		sb.offset = offset
	}
	return sb
}

func (sb *sqlBuilder[T]) Order(orders ...SortFunc) *sqlBuilder[T] {
	if len(orders) == 0 {
		return sb
	}
	for i := range orders {
		orders[i](sb)
	}
	return sb
}

func (sb *sqlBuilder[T]) FromUrlParams(p URLSearchParam) *sqlBuilder[T] {
	return sb.Where(UrlFieldParams(p)).Order(UrlFieldSort(p)).Limit(p.GetLimit()).Offset(p.GetOffset()).Count()
}

func (sb *sqlBuilder[T]) Where(conditions ...Condition) *sqlBuilder[T] {
	if len(conditions) == 0 {
		return sb
	}
	cr := sb.getCriteriaRoot(WhereAnd)
	for i := range conditions {
		conditions[i](cr)
	}
	return sb
}

func (sb *sqlBuilder[T]) Criteria() (b strings.Builder, args []interface{}) {
	if sb.where != nil {
		return sb.where.Build(true)
	}
	return
}

func (sb *sqlBuilder[T]) Select(ctx context.Context, relation string, fields ...string) (count int64, out []T, err error) {
	var rows pgx.Rows
	if sb.rowScanner == nil {
		return 0, nil, errors.New("rowScanner func was undefined")
	}
	if len(fields) > 0 {
		sb.selectFields = fields
	}
	where, args := sb.Criteria()
	if rows, err = sb.dbtx.Query(ctx, sb.selectQueryString(relation, sb.selectFields, where), args...); err != nil {
		return
	}
	defer rows.Close()
	out = make([]T, 0)
	for rows.Next() {
		var i T
		if i, err = sb.rowScanner(rows); err != nil {
			return
		}
		out = append(out, i)
	}
	if err = rows.Err(); err != nil {
		return
	}
	if !sb.count {
		return int64(len(out)), out, nil
	}
	countQuery := fmt.Sprintf("select count(%s) from %s", strings.Join(sb.countFields, ", "), relation)
	if where.Len() > 0 {
		countQuery += " where " + where.String()
	}
	err = sb.dbtx.QueryRow(ctx, countQuery, args...).Scan(&count)
	return
}

func (sb *sqlBuilder[T]) selectQueryString(relation string, fields []string, where strings.Builder) string {
	var query strings.Builder
	query.WriteString(fmt.Sprintf("select %s from %s", strings.Join(fields, ", "), relation))
	if where.Len() > 0 {
		query.WriteString(" where " + where.String())
	}
	if len(sb.orderBy) > 0 {
		query.WriteString(" order by " + strings.Join(sb.orderBy, ", "))
	}
	if sb.limit > 0 {
		query.WriteString(" limit " + fmt.Sprint(sb.limit))
	}
	if sb.offset > 0 {
		query.WriteString(" offset " + fmt.Sprint(sb.offset))
	}
	return query.String()
}

func (sb *sqlBuilder[T]) addFieldSort(sort SortType, fields ...string) {
	if len(fields) == 0 {
		return
	}
	for i := range fields {
		sb.orderBy = append(sb.orderBy, fields[i]+" "+string(sort))
	}
}

func (sb *sqlBuilder[T]) getCriteriaRoot(constraint Constraint) (out *WhereCriteria) {
	if sb.where == nil {
		sb.where = &WhereCriteria{criteria: make(map[Constraint][]*WhereCriteria)}
	}
	if _, ok := sb.where.criteria[constraint]; !ok {
		sb.where.criteria[constraint] = []*WhereCriteria{}
	}
	if len(sb.where.criteria[constraint]) == 0 {
		out = &WhereCriteria{
			constraint: constraint,
			criteria:   map[Constraint][]*WhereCriteria{},
		}
		sb.where.criteria[constraint] = append(sb.where.criteria[constraint], out)
		return
	}
	for i := range sb.where.criteria[constraint] {
		if sb.where.criteria[constraint][i].constraint == constraint {
			return sb.where.criteria[constraint][i]
		}
	}
	var idx = len(sb.where.criteria[constraint])
	sb.where.criteria[constraint] = append(sb.where.criteria[constraint], &WhereCriteria{
		constraint: constraint,
		criteria:   map[Constraint][]*WhereCriteria{},
	})
	return sb.where.criteria[constraint][idx]
}
