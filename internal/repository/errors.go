package repository

import "errors"

var (
	// ErrNotFound is returned when the requested record does not exist.
	ErrNotFound = errors.New("record not found")

	// ErrDuplicateEntry is returned when a unique constraint is violated (e.g. unique email or department name).
	ErrDuplicateEntry = errors.New("record already exists with unique value")

	// ErrConflict is returned when an operation conflicts with existing relational constraints.
	ErrConflict = errors.New("operation conflicts with existing records")
)
