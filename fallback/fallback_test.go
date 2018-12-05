package fallback_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/slok/goresilience"
	"github.com/slok/goresilience/fallback"
)

func TestFallback(t *testing.T) {
	err := errors.New("wanted error")
	tests := []struct {
		name            string
		cmd             goresilience.Func
		fallback        func(called bool) goresilience.Func
		expFallbackCall bool
		expErr          error
	}{
		{
			name: "Fallback should be called in case the runner fails (with no fallback error).",
			cmd: func(ctx context.Context) error {
				return err
			},
			fallback: func(called bool) goresilience.Func {
				return func(context.Context) error {
					called = true
					return nil
				}
			},
			expErr: nil,
		},
		{
			name: "Fallback should be called in case the runner fails (with fallback error).",
			cmd: func(ctx context.Context) error {
				return err
			},

			fallback: func(called bool) goresilience.Func {
				return func(context.Context) error {
					called = true
					return err
				}
			},
			expErr: err,
		},
		{
			name: "Fallback shouldn't be called in case the runner doesn't fail.",
			cmd: func(ctx context.Context) error {
				return nil
			},

			fallback: func(called bool) goresilience.Func {
				return func(context.Context) error {
					called = true
					return nil
				}
			},
			expErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			// Run & check.
			called := false
			runner := fallback.New(test.fallback(called), nil)
			err := runner.Run(context.TODO(), test.cmd)

			if assert.Equal(test.expErr, err) {
				assert.Equal(test.expFallbackCall, called)
			}
		})
	}
}
