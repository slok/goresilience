package fallback

import (
	"context"

	"github.com/slok/goresilience"
)

// New returns a fallback object that will execute the provided
// fallback runner in case the main(r) runner returns an error.
func New(fallback goresilience.Runner, r goresilience.Runner) goresilience.Runner {
	return goresilience.RunnerFunc(func(ctx context.Context) error {
		err := r.Run(ctx)
		if err != nil {
			return fallback.Run(ctx)
		}
		return nil
	})
}
