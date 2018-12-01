package time_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/slok/goresilience/pkg/circuit"
	cbtime "github.com/slok/goresilience/pkg/circuit/time"
)

func TestStaticLatency(t *testing.T) {
	tests := []struct {
		name        string
		timeout     time.Duration
		cmd         circuit.Command
		expFallback bool
		expErr      bool
	}{
		{
			name:    "A command that has been run without timeout shouldn't return a fallback and return the result.",
			timeout: 1 * time.Second,
			cmd: func(ctx context.Context) error {
				return nil
			},
			expFallback: false,
			expErr:      false,
		},
		{
			name:    "A command that has been run without timeout shouldn't return a fallback and return the result (error result).",
			timeout: 1 * time.Second,
			cmd: func(ctx context.Context) error {
				return errors.New("wanted error")
			},
			expFallback: false,
			expErr:      true,
		},
		{
			name:    "A command that has been run with timeout should return a fallback and don't let the function finish and return the err result.",
			timeout: 1,
			cmd: func(ctx context.Context) error {
				time.Sleep(1 * time.Millisecond)
				return errors.New("wanted error")
			},
			expFallback: true,
			expErr:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			fallback, err := cbtime.NewStaticLatency(test.timeout, test.cmd).Run(context.TODO())

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expFallback, fallback)
			}
		})
	}
}
