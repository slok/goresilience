package errors

import "errors"

var (
	// ErrTimeout will be used when a execution timesout.
	ErrTimeout = errors.New("timeout while executing")
	// ErrContextCanceled will be used when the execution has not been executed due to the
	// context cancelation.
	ErrContextCanceled = errors.New("context canceled, logic not executed")
)
