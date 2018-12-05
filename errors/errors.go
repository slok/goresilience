package errors

import "errors"

var (
	// ErrTimeout will be used when a execution timesout.
	ErrTimeout = errors.New("timeout while executing")
	// ErrContextCanceled will be used when the execution has not been executed due to the
	// context cancelation.
	ErrContextCanceled = errors.New("context canceled, logic not executed")
	// ErrTimeoutWaitingForExecution will be used when a exeuction block exceeded waiting
	// to be executed, for example if a worker pool has been busy and the execution object
	// has been waiting to much for being picked by a pool worker.
	ErrTimeoutWaitingForExecution = errors.New("timeout while waiting for execution")
)
