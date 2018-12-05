package timeout

import (
	"context"
	"time"

	"github.com/slok/goresilience"
	"github.com/slok/goresilience/errors"
	runnerutils "github.com/slok/goresilience/internal/util/runner"
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

// NewStatic will wrap a execution unit that will cut the execution of
// a runner when some time passes using the context.
// use 0 timeout for default timeout.
func NewStatic(timeout time.Duration, r goresilience.Runner) goresilience.Runner {
	r = runnerutils.Sanitize(r)

	return goresilience.RunnerFunc(func(ctx context.Context, f goresilience.Func) error {
		// Fallback settings to defaults.
		if timeout == 0 {
			timeout = defaultTimeout
		}

		// Set a timeout to the command using the context.
		// Should we cancel the context if finished...? I guess not, it could continue
		// the middleware chain.
		ctx, _ = context.WithTimeout(ctx, timeout)

		// Run the command
		errc := make(chan error)
		go func() {
			errc <- r.Run(ctx, f)
		}()

		// Wait until the deadline has been reached or w have a result.
		select {
		// Circuit finished correctly.
		case err := <-errc:
			return err
		// Circuit timeout.
		case <-ctx.Done():
			return errors.ErrTimeout
		}
	})
}
