package bulkhead_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/slok/goresilience/bulkhead"
)

func TestStaticBulkheadTimeout(t *testing.T) {
	tests := []struct {
		name          string
		cfg           bulkhead.StaticConfig
		runFunc       func() func(ctx context.Context) error
		timesToCall   int
		expTotalCalls int
		expTotalErrs  int
	}{
		{
			name: "A bulkhead without timeout should complete all runs.",
			cfg:  bulkhead.StaticConfig{},
			runFunc: func() func(ctx context.Context) error {
				return func(ctx context.Context) error {
					time.Sleep(2 * time.Millisecond)
					return nil
				}
			},
			timesToCall:   100,
			expTotalCalls: 100,
			expTotalErrs:  0,
		},
		{
			name: "A bulkhead with timeout should timeout the funcs waiting to run if they have waited too much.",
			cfg: bulkhead.StaticConfig{
				Workers:     10,
				MaxWaitTime: 5 * time.Millisecond,
			},
			runFunc: func() func(ctx context.Context) error {
				return func(ctx context.Context) error {
					time.Sleep(10 * time.Millisecond)
					return nil
				}
			},
			timesToCall:   100,
			expTotalCalls: 10,
			expTotalErrs:  90,
		},
		{
			name: "A bulkhead with timeout should timeout the funcs waiting to run if they have waited too much (allow 2 batches).",
			cfg: bulkhead.StaticConfig{
				Workers:     17,
				MaxWaitTime: 11 * time.Millisecond,
			},
			runFunc: func() func(ctx context.Context) error {
				return func(ctx context.Context) error {
					time.Sleep(10 * time.Millisecond)
					return nil
				}
			},
			timesToCall:   100,
			expTotalCalls: 34,
			expTotalErrs:  66,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			bk := bulkhead.NewStatic(test.cfg, nil)
			results := make(chan error)
			// We call N times using our bulkhead and wait until all have finished.
			for i := 0; i < test.timesToCall; i++ {
				go func() {
					results <- bk.Run(context.TODO(), test.runFunc())
				}()
			}

			// Wait until all calls have finished and count the results
			// if err it means timeout waiting to be executed, if is nil
			// it means it was called and executed successfully.
			gotErrors := 0
			gotCalls := 0
			for i := 0; i < test.timesToCall; i++ {
				err := <-results
				if err != nil {
					gotErrors++
				} else {
					gotCalls++
				}
			}

			// Check total calls and errors.
			assert.Equal(test.expTotalErrs, gotErrors)
		})
	}
}
