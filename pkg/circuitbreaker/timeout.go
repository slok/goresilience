package circuitbreaker

import (
	"context"
	"time"
)

const (
	defaultTimeout = 1 * time.Second
)

// StaticLatency is a circuit breaker that will cut the execution of
// a command when some time passes usint the context.
// The 0 value of the CircuitBreaker is useful.
type StaticLatency struct {
	TimeoutDuration time.Duration
}

// Run satisfies CircuitBreaker interface by executing the command
// and stopping the execution when the timeout passes
func (s StaticLatency) Run(ctx context.Context, cmd Command) (bool, error) {
	if cmd == nil {
		return false, ErrCommandIsNil
	}

	// Fallback settings to defaults.
	if s.TimeoutDuration == 0 {
		s.TimeoutDuration = defaultTimeout
	}

	// Set a timeout to the command using the context.
	ctx, _ = context.WithTimeout(ctx, s.TimeoutDuration)

	// Run the command
	errc := make(chan error)
	go func() {
		errc <- cmd(ctx)
	}()

	// Wait until the deadline has been reached or w have a result.
	select {
	// Circuit finished correctly.
	case err := <-errc:
		return false, err
	// Circuit timeout.
	case <-ctx.Done():
		return true, nil
	}
}
