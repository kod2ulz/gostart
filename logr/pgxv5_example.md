
# Example: Integrating logr with pgx/v5

This example demonstrates how to initialize a `pgx/v5` database connection pool while integrating the application's `logr.Logger` for query logging. This replaces the `logrusadapter` used with `pgx/v4`.

The key is to use the `storage.NewPgxLogger` function provided by the `gostart` library, which creates a `pgx/v5` compatible tracer from your existing logger.

```go
package db

import (
	"context"
	"database/sql"
	"strings"

	notify "github.com/govnet-africa/notify/sql/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kod2ulz/gostart/logr"
	"github.com/kod2ulz/gostart/storage"
	"github.com/kod2ulz/gostart/utils"
	"github.com/pkg/errors"
)

type NotifyQueries struct {
	*notify.Queries
	conf *storage.Conf
}

type SqlDB struct {
	*Queries
	*NotifyQueries
	Conn   *pgxpool.Pool
	conf   *pgxpool.Config
	stConf *storage.Conf
}

func InitSQL(ctx context.Context, log *logr.Logger, conf *storage.Conf) (out *SqlDB, err error) {
	out = &SqlDB{stConf: conf}
	if out.conf, err = pgxpool.ParseConfig(conf.ConnectionString()); err != nil {
		return
	}

	// Use the pgx/v5 tracer from the gostart storage package.
	// This replaces the old logrusadapter.
	out.conf.ConnConfig.Tracer = storage.NewPgxLogger(log)

	// pgxpool.ConnectConfig is deprecated in v5; use pgxpool.NewWithConfig.
	if out.Conn, err = pgxpool.NewWithConfig(ctx, out.conf); err != nil {
		return nil, err
	}

	out.Queries = New(out.Conn)
	out.NotifyQueries = &notify.Queries{Queries: notify.New(out.Conn), conf: conf}
	return
}

func IsSqlNoRows(err error) bool {
	return err != nil && (errors.Is(err, sql.ErrNoRows) || strings.HasSuffix(err.Error(), "no rows in result set"))
}

func SqlTxCopy(db *SqlDB, tx pgx.Tx) *SqlDB {
	return &SqlDB{Queries: New(tx.Conn()), Conn: db.Conn, conf: db.conf}
}

func (db *SqlDB) Transaction(ctx context.Context, fn func(*SqlDB) error) (err error) {
	var tx pgx.Tx
	if tx, err = db.Conn.Begin(ctx); err != nil {
		return errors.Wrap(err, "error initialising transaction")
	}
	defer tx.Rollback(ctx)
	if err = fn(SqlTxCopy(db, tx)); err != nil {
		return errors.Wrap(err, "error executing transaction")
	} else if err = tx.Commit(ctx); err != nil {
		return errors.Wrap(err, "error committing transaction")
	}
	return
}

// ... other functions remain the same ...

```
