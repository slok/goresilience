package chaos_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/slok/goresilience"
	"github.com/slok/goresilience/chaos"
	"github.com/slok/goresilience/errors"
	"github.com/stretchr/testify/assert"
)

var err = fmt.Errorf("wanted error")
var okf = func(ctx context.Context) error { return nil }
var errf = func(ctx context.Context) error { return err }

func TestFailureInjector(t *testing.T) {
	tests := []struct {
		name       string
		cfg        func() chaos.Config
		f          func(runner goresilience.Runner) goresilience.Func
		expTimeout time.Duration
		expErr     error
	}{
		{
			name: "Setting no errors shouldn't return an error.",
			cfg: func() chaos.Config {
				chaosctrl := &chaos.Injector{}
				chaosctrl.SetErrorPercent(0)

				return chaos.Config{
					Injector: chaosctrl,
				}
			},
			f: func(runner goresilience.Runner) goresilience.Func {
				// Make lots of calls to set execution percentage.
				for i := 0; i < 100; i++ {
					runner.Run(context.TODO(), okf)
				}

				return okf
			},
			expErr: nil,
		},
		{
			name: "Setting error percent should make return errors.",
			cfg: func() chaos.Config {
				chaosctrl := &chaos.Injector{}
				chaosctrl.SetErrorPercent(90)

				return chaos.Config{
					Injector: chaosctrl,
				}
			},
			f: func(runner goresilience.Runner) goresilience.Func {
				// Make lots of calls to set execution percentage.
				for i := 0; i < 95; i++ {
					runner.Run(context.TODO(), okf)
				}

				return okf
			},
			expErr: errors.ErrFailureInjected,
		},
		{
			name: "Injecting latency should make the response to be delayed.",
			cfg: func() chaos.Config {
				chaosctrl := &chaos.Injector{}
				chaosctrl.SetLatency(10 * time.Millisecond)

				return chaos.Config{
					Injector: chaosctrl,
				}
			},
			f: func(runner goresilience.Runner) goresilience.Func {
				return okf
			},
			expTimeout: 8 * time.Millisecond,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			cmd := chaos.New(test.cfg(), nil)
			errc := make(chan error)
			go func() {
				errc <- cmd.Run(context.TODO(), test.f(cmd))
			}()

			// If no latency check directly.
			if test.expTimeout == 0 {
				err := <-errc
				assert.Equal(test.expErr, err)
			} else {
				// When latency, check that we timeout, if don't measn that there was no latency.
				select {
				case <-errc:
					assert.Fail("we should timeout due to latency, it didn't, means that the latency didn't work")
				case <-time.After(test.expTimeout):
				}
			}
		})
	}
}
