package query

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Cursor represents a cursor for pagination
type Cursor struct {
	Field string      `json:"field"`
	Value interface{} `json:"value"`
	ID    interface{} `json:"id"`
}

// EncodeCursor encodes a cursor to a base64 string
func EncodeCursor(c Cursor) (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cursor: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// DecodeCursor decodes a base64 cursor string
func DecodeCursor(encoded string) (*Cursor, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode cursor: %w", err)
	}

	var cursor Cursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cursor: %w", err)
	}

	return &cursor, nil
}

// PageInfo contains pagination metadata
type PageInfo struct {
	HasNextPage     bool   `json:"has_next_page"`
	HasPreviousPage bool   `json:"has_previous_page"`
	StartCursor     string `json:"start_cursor,omitempty"`
	EndCursor       string `json:"end_cursor,omitempty"`
	TotalCount      int64  `json:"total_count,omitempty"`
}

// Edge represents an edge in a cursor-based connection
type Edge[T any] struct {
	Node   T      `json:"node"`
	Cursor string `json:"cursor"`
}

// Connection represents a cursor-based connection
type Connection[T any] struct {
	Edges    []Edge[T] `json:"edges"`
	PageInfo PageInfo  `json:"page_info"`
}

// CursorPaginationOptions defines options for cursor-based pagination
type CursorPaginationOptions struct {
	First  *int    // Number of items to fetch (forward pagination)
	After  *string // Cursor to fetch after (forward pagination)
	Last   *int    // Number of items to fetch (backward pagination)
	Before *string // Cursor to fetch before (backward pagination)

	// Sort configuration
	SortField string // Field to sort by (default: "id")
	SortDesc  bool   // Sort in descending order

	// Additional filters
	Where     string
	WhereArgs []interface{}
}

// CursorPaginator provides cursor-based pagination
type CursorPaginator[T any] struct {
	db           *pgxpool.Pool
	table        string
	columns      []string
	idColumn     string
	scanner      func(values []interface{}) (T, error)
	getCursorKey func(T) interface{}
}

// NewCursorPaginator creates a new cursor paginator
func NewCursorPaginator[T any](
	db *pgxpool.Pool,
	table string,
	columns []string,
	idColumn string,
	scanner func(values []interface{}) (T, error),
	getCursorKey func(T) interface{},
) *CursorPaginator[T] {
	return &CursorPaginator[T]{
		db:           db,
		table:        table,
		columns:      columns,
		idColumn:     idColumn,
		scanner:      scanner,
		getCursorKey: getCursorKey,
	}
}

// Paginate performs cursor-based pagination
func (p *CursorPaginator[T]) Paginate(ctx context.Context, opts CursorPaginationOptions) (*Connection[T], error) {
	// Validate options
	if err := p.validateOptions(opts); err != nil {
		return nil, err
	}

	// Set defaults
	if opts.SortField == "" {
		opts.SortField = p.idColumn
	}

	// Determine page size and direction
	pageSize, forward := p.getPageSizeAndDirection(opts)

	// Build the query
	query, args, err := p.buildQuery(opts, pageSize, forward)
	if err != nil {
		return nil, err
	}

	// Execute query
	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute pagination query: %w", err)
	}
	defer rows.Close()

	// Scan results
	var items []T
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		item, err := p.scanner(values)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}

		items = append(items, item)
	}

	// Reverse if backward pagination
	if !forward {
		p.reverseSlice(items)
	}

	// Build connection
	return p.buildConnection(ctx, items, opts, pageSize, forward)
}

// validateOptions validates pagination options
func (p *CursorPaginator[T]) validateOptions(opts CursorPaginationOptions) error {
	if opts.First != nil && opts.Last != nil {
		return fmt.Errorf("cannot specify both 'first' and 'last'")
	}
	if opts.After != nil && opts.Before != nil {
		return fmt.Errorf("cannot specify both 'after' and 'before'")
	}
	if opts.First != nil && *opts.First < 0 {
		return fmt.Errorf("'first' must be non-negative")
	}
	if opts.Last != nil && *opts.Last < 0 {
		return fmt.Errorf("'last' must be non-negative")
	}
	return nil
}

// getPageSizeAndDirection determines page size and direction
func (p *CursorPaginator[T]) getPageSizeAndDirection(opts CursorPaginationOptions) (int, bool) {
	if opts.Last != nil {
		return *opts.Last + 1, false // backward pagination
	}
	if opts.First != nil {
		return *opts.First + 1, true // forward pagination
	}
	return 11, true // default: 10 items + 1 for hasNextPage
}

// buildQuery builds the SQL query for pagination
func (p *CursorPaginator[T]) buildQuery(opts CursorPaginationOptions, pageSize int, forward bool) (string, []interface{}, error) {
	var whereClauses []string
	var args []interface{}
	argNum := 1

	// Add custom where clause
	if opts.Where != "" {
		whereClauses = append(whereClauses, opts.Where)
		args = append(args, opts.WhereArgs...)
		argNum += len(opts.WhereArgs)
	}

	// Add cursor-based filtering
	if forward && opts.After != nil {
		cursor, err := DecodeCursor(*opts.After)
		if err != nil {
			return "", nil, err
		}

		operator := ">"
		if opts.SortDesc {
			operator = "<"
		}

		whereClauses = append(whereClauses, fmt.Sprintf(
			"(%s, %s) %s ($%d, $%d)",
			opts.SortField, p.idColumn, operator, argNum, argNum+1,
		))
		args = append(args, cursor.Value, cursor.ID)
		argNum += 2
	} else if !forward && opts.Before != nil {
		cursor, err := DecodeCursor(*opts.Before)
		if err != nil {
			return "", nil, err
		}

		operator := "<"
		if opts.SortDesc {
			operator = ">"
		}

		whereClauses = append(whereClauses, fmt.Sprintf(
			"(%s, %s) %s ($%d, $%d)",
			opts.SortField, p.idColumn, operator, argNum, argNum+1,
		))
		args = append(args, cursor.Value, cursor.ID)
		argNum += 2
	}

	// Build query
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(p.columns, ", "), p.table)

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Order by
	sortDirection := "ASC"
	if (forward && opts.SortDesc) || (!forward && !opts.SortDesc) {
		sortDirection = "DESC"
	}

	query += fmt.Sprintf(" ORDER BY %s %s, %s %s", opts.SortField, sortDirection, p.idColumn, sortDirection)

	// Limit
	query += fmt.Sprintf(" LIMIT %d", pageSize)

	return query, args, nil
}

// buildConnection builds the connection response
func (p *CursorPaginator[T]) buildConnection(
	ctx context.Context,
	items []T,
	opts CursorPaginationOptions,
	pageSize int,
	forward bool,
) (*Connection[T], error) {
	connection := &Connection[T]{
		Edges:    make([]Edge[T], 0),
		PageInfo: PageInfo{},
	}

	// Determine if there are more pages
	hasMore := len(items) == pageSize
	if hasMore {
		items = items[:len(items)-1]
	}

	// Build edges
	for _, item := range items {
		cursorValue := p.getCursorKey(item)
		cursor := Cursor{
			Field: opts.SortField,
			Value: cursorValue,
			ID:    p.getID(item),
		}

		cursorStr, err := EncodeCursor(cursor)
		if err != nil {
			return nil, err
		}

		connection.Edges = append(connection.Edges, Edge[T]{
			Node:   item,
			Cursor: cursorStr,
		})
	}

	// Set page info
	if len(connection.Edges) > 0 {
		connection.PageInfo.StartCursor = connection.Edges[0].Cursor
		connection.PageInfo.EndCursor = connection.Edges[len(connection.Edges)-1].Cursor
	}

	if forward {
		connection.PageInfo.HasNextPage = hasMore
		connection.PageInfo.HasPreviousPage = opts.After != nil
	} else {
		connection.PageInfo.HasPreviousPage = hasMore
		connection.PageInfo.HasNextPage = opts.Before != nil
	}

	return connection, nil
}

// getID extracts the ID from an item
func (p *CursorPaginator[T]) getID(item T) interface{} {
	// This is a simplified version - in practice, you'd use reflection or a custom function
	return p.getCursorKey(item)
}

// reverseSlice reverses a slice in place
func (p *CursorPaginator[T]) reverseSlice(slice []T) {
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
}

// Helper function to create pointer
func Ptr[T any](v T) *T {
	return &v
}

// Example usage
func ExampleCursorPagination(ctx context.Context, db *pgxpool.Pool) {
	type User struct {
		ID        int64  `json:"id"`
		Name      string `json:"name"`
		CreatedAt string `json:"created_at"`
	}

	paginator := NewCursorPaginator(
		db,
		"users",
		[]string{"id", "name", "created_at"},
		"id",
		func(values []interface{}) (User, error) {
			return User{
				ID:        values[0].(int64),
				Name:      values[1].(string),
				CreatedAt: values[2].(string),
			}, nil
		},
		func(u User) interface{} {
			return u.ID
		},
	)

	// Forward pagination
	opts := CursorPaginationOptions{
		First:     Ptr(10),
		SortField: "created_at",
		SortDesc:  true,
	}

	connection, err := paginator.Paginate(ctx, opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Results: %d items\n", len(connection.Edges))
	fmt.Printf("Has next page: %v\n", connection.PageInfo.HasNextPage)

	// Get next page
	if connection.PageInfo.HasNextPage {
		nextOpts := CursorPaginationOptions{
			First:     Ptr(10),
			After:     &connection.PageInfo.EndCursor,
			SortField: "created_at",
			SortDesc:  true,
		}

		nextConnection, _ := paginator.Paginate(ctx, nextOpts)
		fmt.Printf("Next page: %d items\n", len(nextConnection.Edges))
	}
}
