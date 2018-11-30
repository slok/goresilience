package circuitbreaker

import "errors"

var (
	// ErrCommandIsNil is the error used when a command of a circuitbreaker is nil.
	ErrCommandIsNil = errors.New("command can't be nil")
)
