package circuitbreaker

import (
	"context"
	"sync"
	"time"

	"github.com/slok/goresilience"
	"github.com/slok/goresilience/errors"
	"github.com/slok/goresilience/metrics"
)

type state string

const (
	stateOpen     state = "open"
	stateHalfOpen state = "halfopen"
	stateClosed   state = "closed"
)

// Config is the configuration of the circuit breaker.
type Config struct {
	// ErrorPercentThresholdToOpen is the error percent based on total execution requests
	// to pass from closed to open state.
	ErrorPercentThresholdToOpen int
	// MinimumRequestToOpen is the minimum quantity of execution request needed
	// to evaluate the percent of errors to allow opening the circuit.
	MinimumRequestToOpen int
	// SuccessfulRequiredOnHalfOpen are the number of request (and successes) the
	// circuitbreaker will check when is on half open state before closing the
	// circuit again.
	SuccessfulRequiredOnHalfOpen int
	// WaitDurationInOpenState is how long the circuit will be in
	// open state before moving to half open state.
	WaitDurationInOpenState time.Duration
	// Sliding window settings
	// Example: window size 10 and 1s bucket duration will store the data of the latest 10s
	// to select the state of the circuit.
	//
	// MetricsSlidingWindowBucketQuantity is the number of buckets that will have the window to
	// store the metrics. This window will delete the oldest bucket and create new
	// This way the circuit breaker only uses the latest data to get the state of the circuit.
	MetricsSlidingWindowBucketQuantity int
	// MetricsBucketDuration is the duration for a bucket to store the metrics that collects,
	// This way the circuit will have a window of N buckets of T duration each.
	MetricsBucketDuration time.Duration
}

// defaults will use the default settings from Netflix Hystrix.
func (c *Config) defaults() {
	if c.ErrorPercentThresholdToOpen == 0 {
		c.ErrorPercentThresholdToOpen = 50
	}

	if c.MinimumRequestToOpen == 0 {
		c.MinimumRequestToOpen = 20
	}

	if c.SuccessfulRequiredOnHalfOpen == 0 {
		c.SuccessfulRequiredOnHalfOpen = 1
	}

	if c.WaitDurationInOpenState == 0 {
		c.WaitDurationInOpenState = 5 * time.Second
	}

	if c.MetricsSlidingWindowBucketQuantity == 0 {
		c.MetricsSlidingWindowBucketQuantity = 10
	}

	if c.MetricsBucketDuration == 0 {
		c.MetricsBucketDuration = 1 * time.Second
	}

}

type circuitbreaker struct {
	cfg          Config
	recorder     recorder
	state        state
	stateStarted time.Time
	mu           sync.Mutex
	runner       goresilience.Runner
}

// New returns a new circuit breaker runner.
//
// The circuit breaker has 3 states, close, open and half open.
//
// The circuit starts in closed state, this means that the
// sent funcs will be excuted, the circuit will record the results
// of the executed funcs.
//
// This records will be based on a sliding window divided in buckets
// of a T duration (example, 10 buckets of 1s each, will record the
// results of the last 10s, every second a new bucket will be created
// and the oldest bucket of the 10 buckets will be deleted).
//
// Being in closed state... when the error percent is greater that the
// configured threshold in `ErrorPercentThresholdToOpen` setting
// and at least it made N executions configured in `MinimumRequestToOpen`
// will move to open state.
//
// Being in open state the circuit will return directly an error without
// executing. When the circuit has been in open state for a T duration
// configured in `WaitDurationInOpenState` will move to half open state.
//
// being in half open state... the circuit will allow executing as being
// closed except that the measurements are different, in this case it
// will check that when N executions have been made (configured in
// `SuccessfulRequiredOnHalfOpen`) if all of them have been successfull,
// if all have been ok it will move to closed state, if not it will move
// to open state.
//
// Note: On every state change the recorded metrics will be reset.
func New(cfg Config) goresilience.Runner {
	return NewMiddleware(cfg)(nil)
}

// NewMiddleware returns a middleware with the runner that is return
// by circuitbreaker.New (see that for more information).
func NewMiddleware(cfg Config) goresilience.Middleware {
	cfg.defaults()

	return func(next goresilience.Runner) goresilience.Runner {
		return &circuitbreaker{
			state:        stateClosed,
			recorder:     newBucketWindow(cfg.MetricsSlidingWindowBucketQuantity, cfg.MetricsBucketDuration),
			stateStarted: time.Now(),
			cfg:          cfg,
			runner:       goresilience.SanitizeRunner(next),
		}
	}

}

func (c *circuitbreaker) Run(ctx context.Context, f goresilience.Func) error {
	metricsRecorder, _ := metrics.RecorderFromContext(ctx)

	// Decide state before executing.
	c.preDecideState(metricsRecorder)

	// Execute based on the current state.
	err := c.execute(ctx, f)

	// Measure result.
	c.recorder.inc(err)

	// Decide state after executing.
	c.postDecideState(metricsRecorder)

	return err
}

// preDecideState are the state decision that will be made before the execution. Usually
// this will be executed for the decision state based on time (more than T duration, after T...)
func (c *circuitbreaker) preDecideState(metricsRec metrics.Recorder) {
	state := c.getState()
	switch state {
	case stateOpen:
		// Check if the circuit has been the required time in closed. If yes then
		// we move to half open state.
		if c.sinceStateStart() > c.cfg.WaitDurationInOpenState {
			c.moveState(stateHalfOpen, metricsRec)
		}
	}
}

// postDecideState are the state decision that will be made after the execution. Usually
// this will be executed for the decision state based on measurements (execution errors, totals...)
func (c *circuitbreaker) postDecideState(metricsRec metrics.Recorder) {
	state := c.getState()

	switch state {
	case stateHalfOpen:
		// If we haven't done enough requests in half open then we don't evaluate.
		if c.recorder.totalRequests() >= float64(c.cfg.SuccessfulRequiredOnHalfOpen) {
			state := stateOpen
			// If the requests have been ok then close circuit, if not we should open.
			if c.recorder.errorRate() <= 0 {
				state = stateClosed
			}

			c.moveState(state, metricsRec)
		}
	case stateClosed:
		// Check if we need to go to open state. If we bypassed the thresholds trip the circuit.
		if c.recorder.totalRequests() >= float64(c.cfg.MinimumRequestToOpen) && c.recorder.errorRate() >= float64(c.cfg.MinimumRequestToOpen)/100 {
			c.moveState(stateOpen, metricsRec)
		}
	}

}

func (c *circuitbreaker) execute(ctx context.Context, f goresilience.Func) error {
	state := c.getState()

	// Always execute unless we are on open state.
	switch state {
	case stateOpen:
		return errors.ErrCircuitOpen
	default:
		return c.runner.Run(ctx, f)
	}

}

func (c *circuitbreaker) getState() state {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

func (c *circuitbreaker) sinceStateStart() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Since(c.stateStarted)
}

func (c *circuitbreaker) moveState(state state, metricsRec metrics.Recorder) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Only change if the state changed.
	if c.state != state {
		metricsRec.IncCircuitbreakerState(string(state))

		c.state = state
		c.stateStarted = time.Now()
		c.recorder.reset()
	}
}
