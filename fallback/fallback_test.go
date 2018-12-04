package fallback_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/slok/goresilience/fallback"
	mgoresilience "github.com/slok/goresilience/internal/mocks"
)

func TestFallback(t *testing.T) {
	err := errors.New("wanted error")
	tests := []struct {
		name     string
		runner   func() *mgoresilience.Runner
		fallback func() *mgoresilience.Runner
		expErr   error
	}{
		{
			name: "Fallback should be called in case the runner fails (with no fallback error).",
			runner: func() *mgoresilience.Runner {
				r := &mgoresilience.Runner{}
				r.On("Run", mock.Anything).Return(err)
				return r
			},
			fallback: func() *mgoresilience.Runner {
				// Expect fallback will be called.
				r := &mgoresilience.Runner{}
				r.On("Run", mock.Anything).Return(nil)
				return r
			},
			expErr: nil,
		},
		{
			name: "Fallback should be called in case the runner fails (with fallback error).",
			runner: func() *mgoresilience.Runner {
				r := &mgoresilience.Runner{}
				r.On("Run", mock.Anything).Return(err)
				return r
			},
			fallback: func() *mgoresilience.Runner {
				// Expect fallback will be called.
				r := &mgoresilience.Runner{}
				r.On("Run", mock.Anything).Return(err)
				return r
			},
			expErr: err,
		},
		{
			name: "Fallback shouldn't be called in case the runner doesn't fail.",
			runner: func() *mgoresilience.Runner {
				r := &mgoresilience.Runner{}
				r.On("Run", mock.Anything).Return(nil)
				return r
			},
			fallback: func() *mgoresilience.Runner {
				// Expect fallback not called.
				return &mgoresilience.Runner{}
			},
			expErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			// Mocks.
			mf := test.fallback()
			mr := test.runner()

			// Run & check.
			runner := fallback.New(mf, mr)
			err := runner.Run(context.TODO())

			if assert.Equal(test.expErr, err) {
				mf.AssertExpectations(t)
			}
		})
	}
}
