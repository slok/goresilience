package time

import (
	"context"
	"time"

	"github.com/slok/goresilience/pkg/circuit"
)

const (
	defaultTimeout = 1 * time.Second
)

// result is a internal type used to send circuit breaker results
// using channels.
type result struct {
	fallback bool
	err      error
}

// NewStaticLatency will wrap a circuit braker with a
// circuit breaker that will cut the execution of
// a command when some time passes using the context.
// use 0 timeout for default timeout.
func NewStaticLatency(timeout time.Duration, cb circuit.Breaker) circuit.Breaker {
	return circuit.BreakerFunc(func(ctx context.Context) (bool, error) {
		// Fallback settings to defaults.
		if timeout == 0 {
			timeout = defaultTimeout
		}

		// Set a timeout to the command using the context.
		// Should we cancel the context if finished...? I guess not, it could continue
		// the middleware chain.
		ctx, _ = context.WithTimeout(ctx, timeout)

		// Run the command
		resultc := make(chan result)
		go func() {
			f, err := cb.Run(ctx)
			resultc <- result{fallback: f, err: err}
		}()

		// Wait until the deadline has been reached or w have a result.
		select {
		// Circuit finished correctly.
		case res := <-resultc:
			return res.fallback, res.err
		// Circuit timeout.
		case <-ctx.Done():
			return true, nil
		}
	})
}
