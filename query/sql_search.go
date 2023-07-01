package query

import (
	"context"

	"github.com/kod2ulz/gostart/sqlc"
)

func SearchWithUrlParams[T any](db sqlc.DBTX, ctx context.Context, relation string, params URLSearchParam, rowScanner RowScanFunc[T], fields ...string) (count int64, out []T, err error) {
	return SQLBuilder[T](db, rowScanner).FromUrlParams(params).Select(ctx, relation, fields...)
}