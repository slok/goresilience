package retry_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/slok/goresilience"
	"github.com/slok/goresilience/retry"
)

var err = errors.New("wanted error")

type counterFailer struct {
	notFailOnAttemp int
	timesExecuted   int
}

func (c *counterFailer) Run(ctx context.Context) error {
	c.timesExecuted++
	if c.timesExecuted == c.notFailOnAttemp {
		return nil
	}

	return err
}

func TestRetryResult(t *testing.T) {
	tests := []struct {
		name      string
		cfg       retry.Config
		getF      func() goresilience.Func
		expResult string
		expErr    error
	}{
		{
			name: "A failing execution should not fail if it's retried the required times until returns a non error.",
			cfg: retry.Config{
				WaitBase:       1 * time.Nanosecond,
				DisableBackoff: true,
				Times:          3,
			},
			getF: func() goresilience.Func {
				c := &counterFailer{notFailOnAttemp: 4}
				return c.Run
			},
			expErr: nil,
		},
		{
			name: "A failing execution should fail if it's not retried the required times until returns a non error.",
			cfg: retry.Config{
				WaitBase:       1 * time.Nanosecond,
				DisableBackoff: true,
				Times:          3,
			},
			getF: func() goresilience.Func {
				c := &counterFailer{notFailOnAttemp: 5}
				return c.Run
			},
			expErr: err,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			cmd := retry.New(test.cfg)
			err := cmd.Run(context.TODO(), test.getF())

			assert.Equal(test.expErr, err)
		})
	}
}

var notime = time.Time{}

// patternTimer will store the execution time passed
// (in milliseconds) between the executions.
type patternTimer struct {
	prevExecution time.Time
	waitPattern   []time.Duration
}

func (p *patternTimer) Run(ctx context.Context) error {
	now := time.Now()

	var durationSince time.Duration
	if p.prevExecution != notime {
		durationSince = now.Sub(p.prevExecution)
	}
	p.prevExecution = now

	p.waitPattern = append(p.waitPattern, durationSince.Round(time.Millisecond))

	return errors.New("wanted error")
}

func TestConstantRetry(t *testing.T) {
	tests := []struct {
		name           string
		cfg            retry.Config
		expWaitPattern []time.Duration // use ints so we can used rounded.
	}{
		{
			name: "A retry executions without backoff should be at constant rate. (2ms, 9 retries)",
			cfg: retry.Config{
				WaitBase:       10 * time.Millisecond,
				DisableBackoff: true,
				Times:          4,
			},
			expWaitPattern: []time.Duration{
				0,
				10 * time.Millisecond,
				10 * time.Millisecond,
				10 * time.Millisecond,
				10 * time.Millisecond,
			},
		},
		{
			name: "A retry executions without backoff should be at constant rate (5ms, 4 retries)",
			cfg: retry.Config{
				WaitBase:       30 * time.Millisecond,
				DisableBackoff: true,
				Times:          2,
			},
			expWaitPattern: []time.Duration{
				0,
				30 * time.Millisecond,
				30 * time.Millisecond,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			exec := retry.New(test.cfg)
			pt := &patternTimer{}
			exec.Run(context.TODO(), pt.Run)

			assert.Equal(test.expWaitPattern, pt.waitPattern)
		})
	}
}

func TestBackoffJitterRetry(t *testing.T) {
	tests := []struct {
		name  string
		cfg   retry.Config
		times int
	}{
		{
			name: "Multiple retry executions with backoff should have all different wait times.",
			cfg: retry.Config{
				WaitBase:       50 * time.Millisecond,
				DisableBackoff: false,
				Times:          3,
			},
			times: 3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			occurrences := map[string]struct{}{}

			// Let's do N iterations of the same process.
			for i := 0; i < test.times; i++ {
				runner := &patternTimer{}
				exec := retry.New(test.cfg)
				exec.Run(context.TODO(), runner.Run)

				// Check that the wait pattern results (diferent from 0)
				// are different, this guarantees that at least we are waiting
				// different durations..
				for _, dur := range runner.waitPattern {
					if dur == 0 {
						continue
					}

					// Round to microseconds.
					roundedDur := dur.Round(time.Microsecond)
					key := fmt.Sprintf("%s", roundedDur)
					_, ok := occurrences[key]
					assert.False(ok, "using a exponential jitter a iteration wait time should be different from another, this iteration wait time already appeared (%s)", key)
					occurrences[key] = struct{}{}
				}
			}
		})
	}
}
