package data

import "errors"

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
