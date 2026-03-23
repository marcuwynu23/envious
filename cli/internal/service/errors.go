package service

import "errors"

var (
	// ErrInvalidInput is returned when create/list input is invalid.
	ErrInvalidInput = errors.New("invalid input")
)
