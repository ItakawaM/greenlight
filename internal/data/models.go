package data

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Models struct {
	Movies MovieModelInterface
}

func NewModels(db *pgxpool.Pool) *Models {
	return &Models{
		Movies: &MovieModel{db: db},
	}
}

// handleContextErrors wraps context.DeadlineExceeded for DB timeouts
// and context.Canceled for client cancellations.
func handleContextErrors(err error) error {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return ErrTimeout
	case errors.Is(err, context.Canceled):
		return ErrCanceled
	default:
		return err
	}
}
