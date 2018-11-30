package circuitbreaker

import (
	"context"
)

// Command is the unit of executon of a CircuitBreaker
type Command func(ctx context.Context) error

// CircuitBreaker knows how to execute a command with some safety logic.
type CircuitBreaker interface {
	// Run will run the command using the circuitBreaker.
	Run(ctx context.Context, cmd Command) (fallback bool, err error)
}
