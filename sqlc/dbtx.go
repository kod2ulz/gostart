package sqlc

import (
	"context"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
)

type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}
