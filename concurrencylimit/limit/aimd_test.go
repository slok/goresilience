package limit_test

import (
	"testing"
	"time"

	"github.com/slok/goresilience/concurrencylimit/limit"
	"github.com/stretchr/testify/assert"
)

func TestAIMD(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		cfg      limit.AIMDConfig
		measuref func(alg limit.Limiter)
		expLimit int
	}{
		{
			name: "Starting limit should be the defined on the configuration.",
			cfg: limit.AIMDConfig{
				MinimumLimit: 37,
			},
			measuref: func(alg limit.Limiter) {},
			expLimit: 37,
		},
		{
			name: "At the beggining should increment fast if are too many infligts (increase).",
			cfg: limit.AIMDConfig{
				MinimumLimit:       1,
				SlowStartThreshold: 10,
			},
			measuref: func(alg limit.Limiter) {
				for i := 0; i < 5; i++ {
					alg.MeasureSample(now.Add(-10*time.Millisecond), 0, 30, limit.ResultSuccess)
				}
			},
			expLimit: 6,
		},
		{
			name: "At the beggining should not increment fast if aren't too many infligts (increase).",
			cfg: limit.AIMDConfig{
				MinimumLimit:       1,
				SlowStartThreshold: 10,
			},
			measuref: func(alg limit.Limiter) {
				for i := 0; i < 5; i++ {
					alg.MeasureSample(now.Add(-10*time.Millisecond), 0, 1, limit.ResultSuccess)
				}
			},
			expLimit: 1,
		},
		{
			name: "After passing the threshold it should increase slowly (increase).",
			cfg: limit.AIMDConfig{
				MinimumLimit:       1,
				SlowStartThreshold: 5,
			},
			measuref: func(alg limit.Limiter) {
				for i := 0; i < 20; i++ {
					alg.MeasureSample(now.Add(-10*time.Millisecond), 0, 30, limit.ResultSuccess)
				}
			},
			expLimit: 7,
		},
		{
			name: "With an error it should decrease with a ratio (decrease).",
			cfg: limit.AIMDConfig{
				MinimumLimit:       1,
				SlowStartThreshold: 25,
				BackoffRatio:       0.6,
			},
			measuref: func(alg limit.Limiter) {
				// Start with a high input. (This will set us on a limit of 50)
				for i := 0; i < 1000; i++ {
					alg.MeasureSample(now.Add(-10*time.Millisecond), 0, 3000, limit.ResultSuccess)
				}

				// Fail and make decrease.
				alg.MeasureSample(now.Add(-10*time.Millisecond), 0, 3000, limit.ResultFailure)
			},
			expLimit: 30,
		},
		{
			name: "With a timeout it should decrease with ratio (decrease).",
			cfg: limit.AIMDConfig{
				MinimumLimit:       1,
				SlowStartThreshold: 25,
				BackoffRatio:       0.6,
				RTTTimeout:         500 * time.Millisecond,
			},
			measuref: func(alg limit.Limiter) {
				// Start with a high input. (This will set us on a limit of 50)
				for i := 0; i < 1000; i++ {
					alg.MeasureSample(now.Add(-100*time.Millisecond), 0, 3000, limit.ResultSuccess)
				}

				// Fail and make decrease.
				alg.MeasureSample(now.Add(-1*time.Second), 0, 3000, limit.ResultSuccess)
			},
			expLimit: 30,
		},
		{
			name: "If we decrease to 0, we should stop on the minimum limit.",
			cfg: limit.AIMDConfig{
				MinimumLimit:       3,
				SlowStartThreshold: 25,
				BackoffRatio:       0.6,
			},
			measuref: func(alg limit.Limiter) {
				// Start with a high input. (This will set us on a limit of 50)
				for i := 0; i < 1000; i++ {
					alg.MeasureSample(now.Add(-10*time.Millisecond), 0, 3000, limit.ResultSuccess)
				}

				// Fail and make decrease to the minimum.
				for i := 0; i < 1000; i++ {
					alg.MeasureSample(now.Add(-10*time.Millisecond), 0, 3000, limit.ResultFailure)
				}
			},
			expLimit: 3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			alg := limit.NewAIMD(test.cfg)
			test.measuref(alg)

			assert.Equal(test.expLimit, alg.GetLimit())
		})
	}
}
