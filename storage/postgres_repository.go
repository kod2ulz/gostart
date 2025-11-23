package storage

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository for PostgreSQL
type PostgresRepository[T any, ID comparable] struct {
	db        *pgxpool.Pool
	tableName string
	idColumn  string
	scanner   RowScanner[T]
	cache     CacheStore
	cacheTTL  time.Duration
}

// RowScanner is a function that scans a row into an entity
type RowScanner[T any] func(row pgx.Row) (*T, error)

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository[T any, ID comparable](
	db *pgxpool.Pool,
	tableName string,
	idColumn string,
	scanner RowScanner[T],
) *PostgresRepository[T, ID] {
	return &PostgresRepository[T, ID]{
		db:        db,
		tableName: tableName,
		idColumn:  idColumn,
		scanner:   scanner,
	}
}

// WithCache adds caching support to the repository
func (r *PostgresRepository[T, ID]) WithCache(cache CacheStore, ttl time.Duration) *PostgresRepository[T, ID] {
	r.cache = cache
	r.cacheTTL = ttl
	return r
}

// Create creates a new entity
func (r *PostgresRepository[T, ID]) Create(ctx context.Context, entity *T) error {
	columns, values := r.extractColumnsAndValues(entity)
	placeholders := r.makePlaceholders(len(values))

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		r.tableName,
		strings.Join(columns, ", "),
		placeholders,
	)

	_, err := r.db.Exec(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("failed to create entity: %w", err)
	}

	return nil
}

// Get retrieves an entity by ID
func (r *PostgresRepository[T, ID]) Get(ctx context.Context, id ID) (*T, error) {
	// Check cache first
	if r.cache != nil {
		cacheKey := r.cacheKey(id)
		var entity T
		if err := r.cache.Get(ctx, cacheKey, &entity); err == nil {
			return &entity, nil
		}
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1", r.tableName, r.idColumn)
	row := r.db.QueryRow(ctx, query, id)

	entity, err := r.scanner(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("entity not found")
		}
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	// Cache the result
	if r.cache != nil {
		cacheKey := r.cacheKey(id)
		r.cache.Set(ctx, cacheKey, entity, r.cacheTTL)
	}

	return entity, nil
}

// Update updates an existing entity
func (r *PostgresRepository[T, ID]) Update(ctx context.Context, entity *T) error {
	columns, values := r.extractColumnsAndValues(entity)
	id := r.extractID(entity)

	setClauses := make([]string, len(columns))
	for i, col := range columns {
		setClauses[i] = fmt.Sprintf("%s = $%d", col, i+1)
	}

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s = $%d",
		r.tableName,
		strings.Join(setClauses, ", "),
		r.idColumn,
		len(values)+1,
	)

	values = append(values, id)
	_, err := r.db.Exec(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("failed to update entity: %w", err)
	}

	// Invalidate cache
	if r.cache != nil {
		r.InvalidateCache(ctx, id.(ID))
	}

	return nil
}

// Delete deletes an entity by ID
func (r *PostgresRepository[T, ID]) Delete(ctx context.Context, id ID) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1", r.tableName, r.idColumn)
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	// Invalidate cache
	if r.cache != nil {
		r.InvalidateCache(ctx, id)
	}

	return nil
}

// List retrieves multiple entities with pagination
func (r *PostgresRepository[T, ID]) List(ctx context.Context, opts ListOptions) ([]*T, error) {
	query := fmt.Sprintf("SELECT * FROM %s", r.tableName)

	// Add WHERE clause if filter exists
	if opts.Filter != nil {
		whereClause, args := opts.Filter.ToSQL()
		query += " WHERE " + whereClause
	}

	// Add ORDER BY clause
	if len(opts.Sort) > 0 {
		orderClauses := make([]string, len(opts.Sort))
		for i, sort := range opts.Sort {
			direction := "ASC"
			if sort.Descending {
				direction = "DESC"
			}
			orderClauses[i] = fmt.Sprintf("%s %s", sort.Field, direction)
		}
		query += " ORDER BY " + strings.Join(orderClauses, ", ")
	}

	// Add LIMIT and OFFSET
	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}
	defer rows.Close()

	var entities []*T
	for rows.Next() {
		entity, err := r.scanner(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}
		entities = append(entities, entity)
	}

	return entities, nil
}

// Count returns the total count of entities
func (r *PostgresRepository[T, ID]) Count(ctx context.Context, filter Filter) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", r.tableName)

	if filter != nil {
		whereClause, args := filter.ToSQL()
		query += " WHERE " + whereClause
		var count int64
		err := r.db.QueryRow(ctx, query, args...).Scan(&count)
		return count, err
	}

	var count int64
	err := r.db.QueryRow(ctx, query).Scan(&count)
	return count, err
}

// Exists checks if an entity exists
func (r *PostgresRepository[T, ID]) Exists(ctx context.Context, id ID) (bool, error) {
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE %s = $1)", r.tableName, r.idColumn)
	var exists bool
	err := r.db.QueryRow(ctx, query, id).Scan(&exists)
	return exists, err
}

// WithTransaction executes a function within a transaction
func (r *PostgresRepository[T, ID]) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(ctx); err != nil {
		tx.Rollback(ctx)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// InvalidateCache invalidates the cache for an entity
func (r *PostgresRepository[T, ID]) InvalidateCache(ctx context.Context, id ID) error {
	if r.cache != nil {
		return r.cache.Delete(ctx, r.cacheKey(id))
	}
	return nil
}

// ClearCache clears all cached entities
func (r *PostgresRepository[T, ID]) ClearCache(ctx context.Context) error {
	if r.cache != nil {
		return r.cache.Clear(ctx)
	}
	return nil
}

// Helper methods

func (r *PostgresRepository[T, ID]) cacheKey(id ID) string {
	return fmt.Sprintf("%s:%v", r.tableName, id)
}

func (r *PostgresRepository[T, ID]) extractColumnsAndValues(entity *T) ([]string, []interface{}) {
	v := reflect.ValueOf(entity).Elem()
	t := v.Type()

	var columns []string
	var values []interface{}

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		// Skip unexported fields
		if !value.CanInterface() {
			continue
		}

		// Get column name from tag or use field name
		colName := field.Tag.Get("db")
		if colName == "" {
			colName = strings.ToLower(field.Name)
		}

		// Skip fields marked with "-"
		if colName == "-" {
			continue
		}

		columns = append(columns, colName)
		values = append(values, value.Interface())
	}

	return columns, values
}

func (r *PostgresRepository[T, ID]) extractID(entity *T) interface{} {
	v := reflect.ValueOf(entity).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		colName := field.Tag.Get("db")
		if colName == r.idColumn {
			return value.Interface()
		}
	}

	return nil
}

func (r *PostgresRepository[T, ID]) makePlaceholders(n int) string {
	placeholders := make([]string, n)
	for i := 0; i < n; i++ {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	return strings.Join(placeholders, ", ")
}

// QueryBuilder provides a fluent interface for building queries
type QueryBuilder struct {
	query  string
	args   []interface{}
	argNum int
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		args:   make([]interface{}, 0),
		argNum: 1,
	}
}

// Select starts a SELECT query
func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder {
	if len(columns) == 0 {
		qb.query = "SELECT *"
	} else {
		qb.query = "SELECT " + strings.Join(columns, ", ")
	}
	return qb
}

// From adds a FROM clause
func (qb *QueryBuilder) From(table string) *QueryBuilder {
	qb.query += " FROM " + table
	return qb
}

// Where adds a WHERE clause
func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
	qb.query += " WHERE " + condition
	qb.args = append(qb.args, args...)
	return qb
}

// And adds an AND condition
func (qb *QueryBuilder) And(condition string, args ...interface{}) *QueryBuilder {
	qb.query += " AND " + condition
	qb.args = append(qb.args, args...)
	return qb
}

// Or adds an OR condition
func (qb *QueryBuilder) Or(condition string, args ...interface{}) *QueryBuilder {
	qb.query += " OR " + condition
	qb.args = append(qb.args, args...)
	return qb
}

// OrderBy adds an ORDER BY clause
func (qb *QueryBuilder) OrderBy(columns ...string) *QueryBuilder {
	qb.query += " ORDER BY " + strings.Join(columns, ", ")
	return qb
}

// Limit adds a LIMIT clause
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.query += fmt.Sprintf(" LIMIT %d", limit)
	return qb
}

// Offset adds an OFFSET clause
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.query += fmt.Sprintf(" OFFSET %d", offset)
	return qb
}

// Build returns the final query and arguments
func (qb *QueryBuilder) Build() (string, []interface{}) {
	return qb.query, qb.args
}

// Execute executes the query
func (qb *QueryBuilder) Execute(ctx context.Context, db *pgxpool.Pool) (pgx.Rows, error) {
	return db.Query(ctx, qb.query, qb.args...)
}

// ExecuteScalar executes the query and returns a single value
func (qb *QueryBuilder) ExecuteScalar(ctx context.Context, db *pgxpool.Pool, dest interface{}) error {
	return db.QueryRow(ctx, qb.query, qb.args...).Scan(dest)
}
