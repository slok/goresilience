package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/slok/goresilience"
	"github.com/slok/goresilience/retry"
)

var err = errors.New("error wanted")

type eventuallySucceed struct {
	successfulExecutionAttempt int
	timesExecuted              int
}

func (c *eventuallySucceed) Run(_ context.Context) error {
	c.timesExecuted++
	if c.timesExecuted == c.successfulExecutionAttempt {
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
				c := &eventuallySucceed{successfulExecutionAttempt: 4}
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
				c := &eventuallySucceed{successfulExecutionAttempt: 5}
				return c.Run
			},
			expErr: err,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exec := retry.New(test.cfg)
			err := exec.Run(context.TODO(), test.getF())

			assert.Equal(t, test.expErr, err)
		})
	}
}

var noTime = time.Time{}

// patternTimer will store the execution time passed
// (in milliseconds) between the executions.
type patternTimer struct {
	prevExecution time.Time
	waitPattern   []time.Duration
}

func (p *patternTimer) Run(_ context.Context) error {
	now := time.Now()

	if p.prevExecution == noTime {
		p.prevExecution = now
	} else {
		durationSince := now.Sub(p.prevExecution)
		p.prevExecution = now
		p.waitPattern = append(p.waitPattern, durationSince.Round(time.Millisecond))
	}

	return errors.New("error wanted")
}

func TestConstantRetry(t *testing.T) {
	tests := []struct {
		name           string
		cfg            retry.Config
		expWaitPattern []time.Duration
	}{
		{
			name: "A retry executions without backoff should be at constant rate. (10ms, 4 retries)",
			cfg: retry.Config{
				WaitBase:       10 * time.Millisecond,
				DisableBackoff: true,
				Times:          4,
			},
			expWaitPattern: []time.Duration{
				10 * time.Millisecond,
				10 * time.Millisecond,
				10 * time.Millisecond,
				10 * time.Millisecond,
			},
		},
		{
			name: "A retry executions without backoff should be at constant rate (30ms, 2 retries)",
			cfg: retry.Config{
				WaitBase:       30 * time.Millisecond,
				DisableBackoff: true,
				Times:          2,
			},
			expWaitPattern: []time.Duration{
				30 * time.Millisecond,
				30 * time.Millisecond,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exec := retry.New(test.cfg)
			pt := &patternTimer{}
			_ = exec.Run(context.TODO(), pt.Run)

			assert.Equal(t, test.expWaitPattern, pt.waitPattern)
		})
	}
}

func TestBackoffJitterRetry(t *testing.T) {
	t.Run("Multiple retry executions with backoff should have all different wait times.", func(t *testing.T) {
		// We do several iterations of the same process to reduce the probability that
		// the property we are testing for was obtained by chance.
		const numberOfIterations = 3
		for i := 0; i < numberOfIterations; i++ {
			pt := &patternTimer{}
			exec := retry.New(retry.Config{
				WaitBase:       50 * time.Millisecond,
				DisableBackoff: false,
				Times:          2,
			})
			_ = exec.Run(context.TODO(), pt.Run)
			occurrences := make(map[time.Duration]struct{})

			// Check that the wait pattern results (different from 0)
			// are different, this guarantees that at least we are waiting
			// different durations.
			for _, dur := range pt.waitPattern {
				if dur == 0 {
					continue
				}

				_, ok := occurrences[dur]
				assert.False(t, ok, "Using exponential jitter, attempts' waiting times should be different from one another. This attempt's waiting time has already appeared before (%s)", dur)
				occurrences[dur] = struct{}{}
			}
		}
	})

	t.Run("Multiple retry executions with backoff should have the first waiting time to be no more than the base waiting time configured.", func(t *testing.T) {
		// We do several iterations of the same process to reduce the probability that
		// the property we are testing for was obtained by chance.
		const numberOfIterations = 3
		for i := 0; i < numberOfIterations; i++ {
			pt := &patternTimer{}
			cfg := retry.Config{
				WaitBase:       50 * time.Millisecond,
				DisableBackoff: false,
				Times:          2,
			}
			exec := retry.New(cfg)
			_ = exec.Run(context.TODO(), pt.Run)

			assert.True(t, pt.waitPattern[0] <= cfg.WaitBase)
		}
	})
}
