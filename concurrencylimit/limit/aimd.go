package limit

import (
	"sync"
	"time"
)

// AIMDConfig is the configuration of the algorithm used for the AIMD adaptive limit.
type AIMDConfig struct {
	// MinimumLimit is the nimimum limit the agorithm will decrease. It also will start with this limit.
	MinimumLimit int
	// This is like TCP algorithms `ssthresh`. It will start increasing the limit by one
	// and when reached to this threshold it will change the mode and increase slowly.
	// If set to 0 then slow start will be disabled.
	SlowStartThreshold int
	// RTTTimeout is the rtt is greater than this value it will be measured as a failure.
	RTTTimeout time.Duration
	// BackoffRatio is the ratio used to decrease the limit when a failure occurs.
	// this will be the way is used: new limit = current limit * backoffRatio.
	BackoffRatio float64
	// LimitIncrementInflightFactor will increment the limit only if inflight * LimitIncrementInflightFactor > limit
	LimitIncrementInflightFactor int
}

func (c *AIMDConfig) defaults() {
	// Safety defaults.
	if c.BackoffRatio < 0.5 || c.BackoffRatio > 1 {
		c.BackoffRatio = 0.9
	}

	if c.RTTTimeout == 0 {
		c.RTTTimeout = 2 * time.Second
	}

	if c.MinimumLimit == 0 {
		c.MinimumLimit = 10
	}

	if c.LimitIncrementInflightFactor == 0 {
		c.LimitIncrementInflightFactor = 1
	}
}

// NewAIMD returns a new aimd adaptive Limiter algorithm, based on the TCP congestion algorithm with the same name.
// It increases the limit at a constant rate and when congestion occurs it will decrese by a configured factor.
// More information abouth this algorithm in: https://en.wikipedia.org/wiki/Additive_increase/multiplicative_decrease
func NewAIMD(cfg AIMDConfig) Limiter {
	cfg.defaults()

	return &aimd{
		limit: float64(cfg.MinimumLimit),
		cfg:   cfg,
	}
}

type aimd struct {
	cfg   AIMDConfig
	limit float64
	mu    sync.Mutex
}

// MeasureSample satisfies Algorithm interface.
func (a *aimd) MeasureSample(startTime time.Time, infights int, result Result) int {
	a.mu.Lock()
	defer a.mu.Unlock()

	currentLimit := int(a.limit)
	switch result {
	case ResultSuccess:
		// Although we have a success maybe we are experiencing congestion.
		if time.Since(startTime) > a.cfg.RTTTimeout {
			return a.decreaseLimit()
		}

		// This is a real success.
		// Only increase if we need it. If not we would be increasing forever.
		// If we have double of inflight request waiting then increase.
		if infights > currentLimit*a.cfg.LimitIncrementInflightFactor {
			return a.increaseLimit()
		}

	case ResultFailure:
		return a.decreaseLimit()

	}
	// Same as ignore.
	return currentLimit
}

// decreaseLimit will decrease the limit based on the backoff ratio.
func (a *aimd) decreaseLimit() int {
	a.limit = a.limit * a.cfg.BackoffRatio
	min := float64(a.cfg.MinimumLimit)
	if a.limit <= min {
		a.limit = min
	}
	return int(a.limit)
}

// increaseLimit will increase the limit being aware of slow start.
func (a *aimd) increaseLimit() int {
	// If slowstart disabled or our limit is less than the slow start threshold then
	// increment by one.
	if int(a.limit) < a.cfg.SlowStartThreshold || a.cfg.SlowStartThreshold == 0 {
		a.limit++
	} else {
		// Slow start threshold bypassed.
		a.limit = a.limit + (1 * (1 / a.limit))
	}

	return int(a.limit)
}

// GetLimit satsifies Algorithm interface.
func (a *aimd) GetLimit() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return int(a.limit)
}
