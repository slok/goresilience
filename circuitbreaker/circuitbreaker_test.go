package circuitbreaker_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/slok/goresilience"
	"github.com/slok/goresilience/circuitbreaker"
	"github.com/slok/goresilience/errors"
)

var err = fmt.Errorf("wanted error")
var okf = func(ctx context.Context) error { return nil }
var errf = func(ctx context.Context) error { return err }

func TestCircuitBreaker(t *testing.T) {
	tests := []struct {
		name   string
		cfg    circuitbreaker.Config
		f      func(cb goresilience.Runner) goresilience.Func // Receives the circuit to set the sate in the way we want.
		expErr error
	}{
		{
			name: "The circuit should start in closed state.",
			cfg:  circuitbreaker.Config{},
			f: func(cb goresilience.Runner) goresilience.Func {
				return okf
			},
			expErr: nil,
		},
		{
			name: "After some errors the circuit should be open.",
			cfg: circuitbreaker.Config{
				ErrorPercentThresholdToOpen: 30,
				MinimumRequestToOpen:        10,
			},
			f: func(cb goresilience.Runner) goresilience.Func {
				// Close the circuit first.
				for i := 0; i < 10; i++ {
					cb.Run(context.TODO(), errf)
				}

				return okf
			},
			expErr: errors.ErrCircuitOpen,
		},
		{
			name: "After some errors the circuit should be open, but only if the the maximum of request have been made.",
			cfg: circuitbreaker.Config{
				ErrorPercentThresholdToOpen: 30,
				MinimumRequestToOpen:        11,
			},
			f: func(cb goresilience.Runner) goresilience.Func {
				for i := 0; i < 10; i++ {
					cb.Run(context.TODO(), errf)
				}

				return okf
			},
			expErr: nil,
		},
		{
			name: "A circuit in half open state should admit new requests.",
			cfg: circuitbreaker.Config{
				ErrorPercentThresholdToOpen: 30,
				MinimumRequestToOpen:        10,
				WaitDurationInOpenState:     5 * time.Millisecond,
			},
			f: func(cb goresilience.Runner) goresilience.Func {
				// Close the circuit first.
				for i := 0; i < 10; i++ {
					cb.Run(context.TODO(), errf)
				}

				// Wait the circuit in open state to go in half open state.
				time.Sleep(6 * time.Millisecond)

				return okf
			},
			expErr: nil,
		},
		{
			name: "A circuit in half open state should return the execution result while it's on this phase.",
			cfg: circuitbreaker.Config{
				ErrorPercentThresholdToOpen: 30,
				MinimumRequestToOpen:        10,
				WaitDurationInOpenState:     5 * time.Millisecond,
			},
			f: func(cb goresilience.Runner) goresilience.Func {
				// Close the circuit first.
				for i := 0; i < 10; i++ {
					cb.Run(context.TODO(), errf)
				}

				// Wait the circuit in open state to go in half open state.
				time.Sleep(6 * time.Millisecond)

				return errf
			},
			expErr: err,
		},
		{
			name: "A circuit in half open state should open the circuit if the execution errors.",
			cfg: circuitbreaker.Config{
				ErrorPercentThresholdToOpen: 30,
				MinimumRequestToOpen:        10,
				WaitDurationInOpenState:     5 * time.Millisecond,
			},
			f: func(cb goresilience.Runner) goresilience.Func {
				// Close the circuit first.
				for i := 0; i < 10; i++ {
					cb.Run(context.TODO(), errf)
				}

				// Wait the circuit in open state to go in half open state.
				time.Sleep(6 * time.Millisecond)

				// Trigger from half open to open again.
				cb.Run(context.TODO(), errf)

				return errf
			},
			expErr: errors.ErrCircuitOpen,
		},
		{
			name: "A circuit in half open state should close the circuit if the execution success.",
			cfg: circuitbreaker.Config{
				ErrorPercentThresholdToOpen: 30,
				MinimumRequestToOpen:        10,
				WaitDurationInOpenState:     5 * time.Millisecond,
			},
			f: func(cb goresilience.Runner) goresilience.Func {
				// Close the circuit first.
				for i := 0; i < 10; i++ {
					cb.Run(context.TODO(), errf)
				}

				// Wait the circuit in open state to go in half open state.
				time.Sleep(6 * time.Millisecond)

				// Trigger from half open to close.
				cb.Run(context.TODO(), okf)

				return okf
			},
			expErr: nil,
		},
		{
			name: "The sliding window should forget the old recorded settings (short window enought to forget).",
			cfg: circuitbreaker.Config{
				ErrorPercentThresholdToOpen:        30,
				MinimumRequestToOpen:               10,
				SuccessfulRequiredOnHalfOpen:       1,
				WaitDurationInOpenState:            5 * time.Millisecond,
				MetricsSlidingWindowBucketQuantity: 5,
				MetricsBucketDuration:              5 * time.Millisecond,
			},
			f: func(cb goresilience.Runner) goresilience.Func {
				// Close the circuit first.
				for i := 0; i < 10; i++ {
					cb.Run(context.TODO(), errf)
				}

				// Wait the circuit in open state to go in half open state.
				time.Sleep(6 * time.Millisecond)

				// Open the circuit with lots of good requests.
				for i := 0; i < 100; i++ {
					cb.Run(context.TODO(), okf)
				}

				// Wait to reset the window.
				time.Sleep(30 * time.Millisecond)

				// Try to close again with the same number of errors.
				// This should closed, this means that the 100 good
				// request that we made before don't count for the
				// percent rate threshold. So we can assure that the
				// sliding window has worked and it forgot the previous
				// metrics.
				for i := 0; i < 10; i++ {
					cb.Run(context.TODO(), errf)
				}

				cb.Run(context.TODO(), okf)
				return okf
			},
			expErr: errors.ErrCircuitOpen,
		},
		{
			name: "The sliding window should forget the old recorded settings (big window not enought to forget).",
			cfg: circuitbreaker.Config{
				ErrorPercentThresholdToOpen:        30,
				MinimumRequestToOpen:               10,
				SuccessfulRequiredOnHalfOpen:       1,
				WaitDurationInOpenState:            5 * time.Millisecond,
				MetricsSlidingWindowBucketQuantity: 100,
				MetricsBucketDuration:              5 * time.Millisecond,
			},
			f: func(cb goresilience.Runner) goresilience.Func {
				// Close the circuit first.
				for i := 0; i < 10; i++ {
					cb.Run(context.TODO(), errf)
				}

				// Wait the circuit in open state to go in half open state.
				time.Sleep(6 * time.Millisecond)

				// Open the circuit with lots of good requests.
				for i := 0; i < 100; i++ {
					cb.Run(context.TODO(), okf)
				}

				// Wait to reset the window
				time.Sleep(30 * time.Millisecond)

				// Try to close again with the same number of errors.
				// This should not close, this means that the 100 good
				// request that we made before are on the record regsitry
				// and count for the percent threshold. So we can assure
				// that the sliding window has worked and it dind't forgot
				// the previous metrics because it didn't slide yet.
				for i := 0; i < 10; i++ {
					cb.Run(context.TODO(), errf)
				}

				cb.Run(context.TODO(), okf)
				return okf
			},
			expErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			cb := circuitbreaker.New(test.cfg)
			err := cb.Run(context.TODO(), test.f(cb))

			assert.Equal(test.expErr, err)
		})
	}
}

func BenchmarkCircuitBreaker(b *testing.B) {
	b.StopTimer()

	benchs := []struct {
		name string
		f    func(cb goresilience.Runner) goresilience.Func
		cfg  circuitbreaker.Config
	}{
		{
			name: "benchmark with default settings.",
			f: func(cb goresilience.Runner) goresilience.Func {
				return errf
			},
			cfg: circuitbreaker.Config{},
		},
	}

	for _, bench := range benchs {
		b.Run(bench.name, func(b *testing.B) {
			// Prepare.
			cb := circuitbreaker.New(bench.cfg)
			f := bench.f(cb)
			// execute the benhmark.
			for n := 0; n < b.N; n++ {
				b.StartTimer()
				cb.Run(context.TODO(), f)
				b.StopTimer()
			}
		})
	}
}
