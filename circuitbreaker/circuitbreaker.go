package circuitbreaker

import (
	"context"
	"sync"
	"time"

	"github.com/slok/goresilience"
	"github.com/slok/goresilience/errors"
	runnerutils "github.com/slok/goresilience/internal/util/runner"
)

type state int

const (
	stateOpen state = iota
	stateHalfOpen
	stateClosed
)

// Config is the configuration of the circuit breaker.
type Config struct {
	// ErrorPercentThresholdToOpen is the minimum percentage of errors where the circuit
	// will pass to open phase.
	ErrorPercentThresholdToOpen int
	// MinimumRequestToOpen is the minimum quantity of execution reuqest needed
	// to evaluate the percent of errors to allow opening the circuit.
	MinimumRequestToOpen int

	// SuccessRequiredOnHalfOpen are the number of continous successes the
	// circuitbreaker will check when is on halfopen state before closing the
	// circuit again.
	SuccessRequiredOnHalfOpen int

	// WaitDurationInOpenState is the duration the circuit will be in
	// open state before moving to half open state.
	WaitDurationInOpenState time.Duration
}

func (c *Config) defaults() {
	if c.WaitDurationInOpenState == 0 {
		c.WaitDurationInOpenState = 5 * time.Second
	}

	if c.ErrorPercentThresholdToOpen == 0 {
		c.ErrorPercentThresholdToOpen = 50
	}

	if c.MinimumRequestToOpen == 0 {
		c.MinimumRequestToOpen = 20
	}

	if c.SuccessRequiredOnHalfOpen == 0 {
		c.SuccessRequiredOnHalfOpen = 1
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
// TODO: explanation.
func New(cfg Config, r goresilience.Runner) goresilience.Runner {
	cfg.defaults()

	return &circuitbreaker{
		state:        stateClosed,
		recorder:     &counter{},
		stateStarted: time.Now(),
		cfg:          cfg,
		runner:       runnerutils.Sanitize(r),
	}
}

func (c *circuitbreaker) Run(ctx context.Context, f goresilience.Func) error {
	// Decide state before executing.
	c.preDecideState()

	// Execute based on the current state.
	err := c.execute(ctx, f)

	// Measure result.
	c.recorder.inc(err)

	// Decide state after executing.
	c.postDecideState()

	return err
}

// preDecideState are the state decision that will be made before the execution. Usually
// this will be executed for the decision state based on time (more than T duration, after T...)
func (c *circuitbreaker) preDecideState() {
	state := c.getState()
	switch state {
	case stateOpen:
		// Check if the circuit has been the required time in closed. If yes then
		// we move to half open state.
		if c.sinceStateStart() > c.cfg.WaitDurationInOpenState {
			c.moveState(stateHalfOpen)
		}
	}
}

// postDecideState are the state decision that will be made after the execution. Usually
// this will be executed for the decision state based on measurements (execution errors, totals...)
func (c *circuitbreaker) postDecideState() {
	state := c.getState()

	switch state {
	case stateHalfOpen:
		// If we haven't done enough requests in half open then we don't evaluate.
		if c.recorder.totalRequests() >= float64(c.cfg.SuccessRequiredOnHalfOpen) {
			state := stateOpen
			// If the requests have been ok then close circuit, if not we should open.
			if c.recorder.errorRate() <= 0 {
				state = stateClosed
			}

			c.moveState(state)
		}
	case stateClosed:
		// Check if we need to go to open state. If we bypassed the thresholds trip the circuit.
		if c.recorder.totalRequests() >= float64(c.cfg.MinimumRequestToOpen) && c.recorder.errorRate() >= float64(c.cfg.MinimumRequestToOpen)/100 {
			c.moveState(stateOpen)
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

func (c *circuitbreaker) moveState(state state) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Only change if the state changed.
	if c.state != state {
		c.state = state
		c.stateStarted = time.Now()
		c.recorder.reset()
	}
}
