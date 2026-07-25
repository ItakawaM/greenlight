package data

import (
	"context"
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	// ErrRecordNotFound is returned when a database query finds no matching rows.
	ErrRecordNotFound = errors.New("record not found")

	// ErrEditConflict is returned when fails its optimistic concurrency check.
	ErrEditConflict = errors.New("edit conflict")

	// ErrTimeout is returned when a database query does not complete
	// before its context deadline elapses.
	ErrTimeout = errors.New("db query timeout")

	// ErrCanceled is returned when a database query's context is canceled before completion.
	// Example: client disconnected.
	ErrCanceled = errors.New("db query canceled")

	// ErrDuplicateEmail is returned during user creation when a user with the provided email
	// already exists.
	ErrDuplicateEmail = errors.New("duplicate email")
)

// isUniqueViolationError checks whether the provided pgx error
// is a unique violation of the provided constraint.
func isUniqueViolationError(err error, constraint string) bool {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		if pgErr.Code == pgerrcode.UniqueViolation && pgErr.ConstraintName == constraint {
			return true
		}
	}

	return false
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
