package concurrencylimit_test

import (
	"context"
	"testing"

	"github.com/slok/goresilience/concurrencylimit"
	"github.com/slok/goresilience/concurrencylimit/limit"
	mexecute "github.com/slok/goresilience/internal/mocks/concurrencylimit/execute"
	mlimit "github.com/slok/goresilience/internal/mocks/concurrencylimit/limit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var fOK = func(_ context.Context) error { return nil }
var successPolicy = func(ctx context.Context, err error) limit.Result { return limit.ResultSuccess }

func TestConcurrencyLimit(t *testing.T) {
	tests := []struct {
		name   string
		policy concurrencylimit.ExecutionResultPolicy
		mock   func(ml *mlimit.Limiter, me *mexecute.Executor)
		expErr error
	}{
		{
			name:   "Executing a Func should  measure the samples with the policy result and set the worker quantity based on the limiter limit.",
			policy: successPolicy,
			mock: func(ml *mlimit.Limiter, me *mexecute.Executor) {
				measuredLimit := 97

				// Return ok on execution.
				me.On("Execute", mock.Anything).Once().Return(nil)

				// Expect measuring the sample after the execution.
				ml.On("MeasureSample", mock.Anything, mock.Anything, 0, limit.ResultSuccess).Once().Return(measuredLimit)

				// Expect setting the limit after measuring with the limiter.
				me.On("SetWorkerQuantity", measuredLimit)

			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			// Mocks.
			ml := &mlimit.Limiter{}
			me := &mexecute.Executor{}
			// First limit get/set on creation (prepare).
			ml.On("GetLimit").Once().Return(0)
			me.On("SetWorkerQuantity", 0).Once()
			test.mock(ml, me)

			cfg := concurrencylimit.Config{
				Limiter:  ml,
				Executor: me,
			}

			runner := concurrencylimit.New(cfg)
			err := runner.Run(context.TODO(), fOK)

			assert.Equal(test.expErr, err)
		})
	}
}
