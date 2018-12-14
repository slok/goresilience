package main

import (
	"context"
	"fmt"
	"log"

	"github.com/slok/goresilience"
	"github.com/slok/goresilience/retry"
)

// Config is the configuration of constFailer
type Config struct {
	// FailEveryTimes will make the runner return an error every N executed times.
	FailEveryTimes int
}

// NewFailer is like NewFailerMiddleware but will not wrap any other runner, is standalone.
func NewFailer(cfg Config) goresilience.Runner {
	return NewFailerMiddleware(cfg)(nil)
}

// NewFailerMiddleware returns a new middleware that will wrap runners and will fail
// evey N times of executions.
func NewFailerMiddleware(cfg Config) goresilience.Middleware {
	return func(next goresilience.Runner) goresilience.Runner {
		calledTimes := 0
		// Use the RunnerFunc helper so we don't need to create a new type.
		return goresilience.RunnerFunc(func(ctx context.Context, f goresilience.Func) error {
			// We should lock the counter writes, not made because this is an example.
			calledTimes++

			if calledTimes == cfg.FailEveryTimes {
				calledTimes = 0
				return fmt.Errorf("failed due to %d call", calledTimes)
			}

			// Run using the the chain.
			next = goresilience.SanitizeRunner(next)
			return next.Run(ctx, f)
		})
	}
}

func main() {
	failerCfg := Config{
		FailEveryTimes: 2,
	}

	// Use it standalone.
	//cmd := NewFailer(failerCfg)

	// Or... create our execution chain.
	retrier := retry.NewMiddleware(retry.Config{})
	failer := NewFailerMiddleware(failerCfg)
	cmd := goresilience.RunnerChain(retrier, failer)

	for i := 0; i < 200; i++ {
		// Execute.
		result := ""
		err := cmd.Run(context.TODO(), func(_ context.Context) error {
			result = "all ok"
			return nil
		})

		if err != nil {
			result = "not ok, but fallback"
		}

		log.Printf("the result is: %s", result)
	}
}
