package goresilience

import (
	"context"

	"github.com/slok/goresilience/errors"
)

// Runner knows how to execute a execution logic and returns error if errors.
type Runner interface {
	// Run will run the unit of execution.
	Run(ctx context.Context) error
}

// RunnerFunc is a helper that will satisfies circuit.Breaker interface by using a function.
type RunnerFunc func(ctx context.Context) error

// Run satisfies Runner interface.
func (f RunnerFunc) Run(ctx context.Context) error {
	// Only execute if we reached to the execution and the context has not been cancelled.
	select {
	case <-ctx.Done():
		return errors.ErrContextCanceled
	default:
		return f(ctx)
	}
}
