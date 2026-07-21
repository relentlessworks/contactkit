package store

import "errors"

var (
	// ErrNotFound is returned when a record is not found.
	ErrNotFound = errors.New("record not found")
	// ErrAlreadyExists is returned when a record already exists.
	ErrAlreadyExists = errors.New("record already exists")
)
