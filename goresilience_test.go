package goresilience_test

import (
	"context"
	"testing"

	"github.com/slok/goresilience"
	"github.com/stretchr/testify/assert"
)

type spy struct {
	next   goresilience.Runner
	called bool
}

func (s *spy) Run(ctx context.Context, f goresilience.Func) error {
	s.called = true
	return s.next.Run(ctx, f)
}

func newSpyMiddleware(spy *spy) goresilience.Middleware {
	return func(next goresilience.Runner) goresilience.Runner {
		spy.next = next
		return spy
	}
}

func TestRunnerChain(t *testing.T) {
	tests := []struct {
		name    string
		runners int
	}{
		{
			name:    "A chain of 100 runners should call all of them",
			runners: 5,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			// Create the middleware of runners.
			spies := []*spy{}
			middlewares := []goresilience.Middleware{}

			for i := 0; i < test.runners; i++ {
				spy := &spy{}
				spies = append(spies, spy)
				middlewares = append(middlewares, newSpyMiddleware(spy))
			}

			// Call all our chain.
			runner := goresilience.RunnerChain(middlewares...)
			runner.Run(context.TODO(), func(ctx context.Context) error {
				return nil
			})

			// Check all were called.
			for _, spy := range spies {
				assert.True(spy.called)
			}
		})
	}
}
