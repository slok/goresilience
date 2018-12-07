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

// StaticConfig is the configuration of the Static timeout.
type StaticConfig struct {
	Timeout time.Duration
}

func (s *StaticConfig) defaults() {
	if s.Timeout <= 0 {
		s.Timeout = defaultTimeout
	}
}

// result is a internal type used to send circuit breaker results
// using channels.
type result struct {
	fallback bool
	err      error
}

// NewStatic will wrap a execution unit that will cut the execution of
// a runner when some time passes using the context.
// use 0 timeout for default timeout.
func NewStatic(cfg StaticConfig, r goresilience.Runner) goresilience.Runner {
	cfg.defaults()

	r = runnerutils.Sanitize(r)

	return goresilience.RunnerFunc(func(ctx context.Context, f goresilience.Func) error {
		// Set a timeout to the command using the context.
		// Should we cancel the context if finished...? I guess not, it could continue
		// the middleware chain.
		ctx, _ = context.WithTimeout(ctx, cfg.Timeout)

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
