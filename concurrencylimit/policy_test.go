package concurrencylimit_test

import (
	"context"
	"errors"
	"testing"

	"github.com/slok/goresilience/concurrencylimit"
	"github.com/slok/goresilience/concurrencylimit/limit"
	goresilienceerrors "github.com/slok/goresilience/errors"
	"github.com/stretchr/testify/assert"
)

func TestPolicies(t *testing.T) {
	tests := []struct {
		name      string
		policy    concurrencylimit.ExecutionResultPolicy
		err       error
		expResult limit.Result
	}{
		{
			name:      "FailureOnExternalErrorPolicy no failure should return success",
			policy:    concurrencylimit.FailureOnExternalErrorPolicy,
			err:       nil,
			expResult: limit.ResultSuccess,
		},
		{
			name:      "FailureOnExternalErrorPolicy with an external failure should return a failure",
			policy:    concurrencylimit.FailureOnExternalErrorPolicy,
			err:       errors.New("external error"),
			expResult: limit.ResultFailure,
		},
		{
			name:      "FailureOnExternalErrorPolicy with a exec rejection failure should return ignore",
			policy:    concurrencylimit.FailureOnExternalErrorPolicy,
			err:       goresilienceerrors.ErrRejectedExecution,
			expResult: limit.ResultIgnore,
		},
		{
			name:      "NoFailurePolicy no failure should return success",
			policy:    concurrencylimit.NoFailurePolicy,
			err:       nil,
			expResult: limit.ResultSuccess,
		},
		{
			name:      "NoFailurePolicy with any failure should return ignore",
			policy:    concurrencylimit.NoFailurePolicy,
			err:       errors.New("external error"),
			expResult: limit.ResultIgnore,
		},
		{
			name:      "FailureOnRejectedPolicy no failure should return success",
			policy:    concurrencylimit.FailureOnRejectedPolicy,
			err:       nil,
			expResult: limit.ResultSuccess,
		},
		{
			name:      "FailureOnRejectedPolicy with an external failure should return ignore",
			policy:    concurrencylimit.FailureOnRejectedPolicy,
			err:       errors.New("external error"),
			expResult: limit.ResultIgnore,
		},
		{
			name:      "FailureOnRejectedPolicy with a exec rejection failure should return a failure",
			policy:    concurrencylimit.FailureOnRejectedPolicy,
			err:       goresilienceerrors.ErrRejectedExecution,
			expResult: limit.ResultFailure,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			res := test.policy(context.TODO(), test.err)
			assert.Equal(test.expResult, res)
		})
	}
}
