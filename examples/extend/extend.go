package main

import (
	"context"
	"fmt"
	"log"

	"github.com/slok/goresilience"
	runnerutils "github.com/slok/goresilience/internal/util/runner"
	"github.com/slok/goresilience/retry"
)

// Config is the configuration of constFailer
type Config struct {
	// FailEveryTimes will make the runner return an error every N executed times.
	FailEveryTimes int
}

// NewFailer returns a new runner that will fail evey N times of executions.
func NewFailer(cfg Config, r goresilience.Runner) goresilience.Runner {
	r = runnerutils.Sanitize(r)

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
		return r.Run(ctx, f)
	})
}

func main() {
	// Create our execution chain (nil marks the end of the chain).
	cmd := retry.New(retry.Config{},
		NewFailer(Config{
			FailEveryTimes: 2,
		}, nil))

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
