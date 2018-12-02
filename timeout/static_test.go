package timeout_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	grerrors "github.com/slok/goresilience/errors"
	"github.com/slok/goresilience/pkg/circuit"
	"github.com/slok/goresilience/timeout"
)

func TestStaticLatency(t *testing.T) {
	err := errors.New("wanted error")

	tests := []struct {
		name    string
		timeout time.Duration
		cmd     circuit.Command
		expErr  error
	}{
		{
			name:    "A command that has been run without timeout shouldn't return and error.",
			timeout: 1 * time.Second,
			cmd: func(ctx context.Context) error {
				return nil
			},
			expErr: nil,
		},
		{
			name:    "A command that has been run without timeout should return aerror result).",
			timeout: 1 * time.Second,
			cmd: func(ctx context.Context) error {
				return err
			},
			expFallback: false,
			expErr:      err,
		},
		{
			name:    "A command that has been run with timeout should return a fallback and don't let the function finish and return the err result.",
			timeout: 1,
			cmd: func(ctx context.Context) error {
				time.Sleep(1 * time.Millisecond)
				return errors.New("wanted error")
			},
			expFallback: true,
			expErr:      grerrors.ErrTimeout,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			err := timeout.NewStatic(test.timeout, test.cmd).Run(context.TODO())
			assert.Equal(test.expErr, err)
		})
	}
}
