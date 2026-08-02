package data

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX is satisfied by both *pgxpool.Pool and pgx.Tx, letting models
// run against either a connection pool or an active transaction.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Models represents a wrapped struct around all DB models.
type Models struct {
	Movies MovieModelInterface
	Users  UserModelInterface
	Tokens TokenModelInterface
	db     *pgxpool.Pool
}

func newModels(db DBTX, pool *pgxpool.Pool) *Models {
	return &Models{
		Movies: &MovieModel{db: db},
		Users:  &UserModel{db: db},
		Tokens: &TokenModel{db: db},
		db:     pool,
	}
}

// NewModels creates a new Models instance with all the implemented DB models.
func NewModels(pool *pgxpool.Pool) *Models {
	return newModels(pool, pool)
}

// WithTx runs fn inside a transaction. It commits if fn returns nil,
// and rolls back otherwise (or if fn panics).
func (m *Models) WithTx(ctx context.Context, fn func(*Models) error) error {
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := fn(newModels(tx, m.db)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
