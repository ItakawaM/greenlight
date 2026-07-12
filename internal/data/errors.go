package data

import "errors"

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")

	ErrTimeout  = errors.New("db query timeout")
	ErrCanceled = errors.New("db query canceled")
)
