package fallback

import (
	"context"

	"github.com/slok/goresilience"
	runnerutils "github.com/slok/goresilience/internal/util/runner"
)

// New returns a fallback object that will execute the provided
// fallback runner in case the main(r) runner returns an error.
func New(fallback goresilience.Func, r goresilience.Runner) goresilience.Runner {
	r = runnerutils.Sanitize(r)

	return goresilience.RunnerFunc(func(ctx context.Context, f goresilience.Func) error {
		err := r.Run(ctx, f)
		if err != nil {
			return fallback(ctx)
		}
		return nil
	})
}
