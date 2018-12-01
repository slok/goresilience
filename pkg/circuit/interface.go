package circuit

import (
	"context"
)

// Breaker knows how to execute and break the circuit or execution and returns
// if it should fallback or error.
type Breaker interface {
	// Run will run the unit of execution and return fallback in case it should fallback
	// breaking the excution if it needs to break it and returning error or not if something has
	// gone wrong or should notified that errored.
	Run(ctx context.Context) (fallback bool, err error)
}

// BreakerFunc is a helper that will satisfies circuit.Breaker interface by using a function.
type BreakerFunc func(ctx context.Context) (fallback bool, err error)

// Run satisfies circuit.Breaker interface.
func (f BreakerFunc) Run(ctx context.Context) (fallback bool, err error) {
	return f(ctx)
}
