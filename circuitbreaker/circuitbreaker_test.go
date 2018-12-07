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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			cb := circuitbreaker.New(test.cfg, nil)
			err := cb.Run(context.TODO(), test.f(cb))

			assert.Equal(test.expErr, err)
		})
	}
}
