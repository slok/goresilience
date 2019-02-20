package concurrencylimit

import (
	"context"

	"github.com/slok/goresilience/concurrencylimit/limit"
	"github.com/slok/goresilience/errors"
)

// ExecutionResultPolicy is the function that will have the responsibility of
// categorizing the result of the execution for the limiter algorithm. For example
// depending on the type of the execution a connection error could be treated
// like an failure in the algorithm or just ignore it.
type ExecutionResultPolicy func(ctx context.Context, err error) limit.Result

// FailureOnExternalErrorPolicy will treat as failure every error that is not
// from concurrencylimit package (this is the error by the limiters).
var FailureOnExternalErrorPolicy = func(_ context.Context, err error) limit.Result {
	// Everything ok.
	if err == nil {
		return limit.ResultSuccess
	}

	// Our own failures should be ignored, the rest nope.
	if err != nil && err != errors.ErrRejectedExecution {
		return limit.ResultFailure
	}

	return limit.ResultIgnore
}

// NoFailurePolicy will treat will never return a failure, just ignore when an error
// occurs, this can be used to adapt only on RTT/latency.
var NoFailurePolicy = func(_ context.Context, err error) limit.Result {
	// Everything ok.
	if err == nil {
		return limit.ResultSuccess
	}

	return limit.ResultIgnore
}

// FailureOnRejectedPolicy will treat as failure every time the execution has been
// rejected with a `errors.ErrRejectedExecution` error.
var FailureOnRejectedPolicy = func(_ context.Context, err error) limit.Result {
	// Everything ok.
	if err == nil {
		return limit.ResultSuccess
	}

	if err != nil && err == errors.ErrRejectedExecution {
		return limit.ResultFailure
	}

	return limit.ResultIgnore
}
