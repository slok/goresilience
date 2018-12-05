package goresilience

import (
	"context"

	"github.com/slok/goresilience/errors"
)

// Func is the function to execute with resilience.
type Func func(ctx context.Context) error

// Command is the unit of execution.
type Command struct{}

// Run satisfies Runner interface.
func (Command) Run(ctx context.Context, f Func) error {
	// Only execute if we reached to the execution and the context has not been cancelled.
	select {
	case <-ctx.Done():
		return errors.ErrContextCanceled
	default:
		return f(ctx)
	}
}

// Runner knows how to execute a execution logic and returns error if errors.
type Runner interface {
	// Run will run the unit of execution passed on f.
	Run(ctx context.Context, f Func) error
}

// RunnerFunc is a helper that will satisfies circuit.Breaker interface by using a function.
type RunnerFunc func(ctx context.Context, f Func) error

// Run satisfies Runner interface.
func (r RunnerFunc) Run(ctx context.Context, f Func) error {
	// Only execute if we reached to the execution and the context has not been cancelled.
	select {
	case <-ctx.Done():
		return errors.ErrContextCanceled
	default:
		return r(ctx, f)
	}
}
