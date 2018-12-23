package concurrencylimit

import (
	"context"
	"sync"
	"time"

	"github.com/slok/goresilience"
	"github.com/slok/goresilience/concurrencylimit/execute"
	"github.com/slok/goresilience/concurrencylimit/limit"
	"github.com/slok/goresilience/errors"
	"github.com/slok/goresilience/metrics"
)

// ExecutionResultPolicy is the function that will have the responsibility of
// categorizing the result of the execution for the limit algorithm. For example
// depending on the type of the execution a connection error could be treated
// like an failure in the algorithm or just ignore it.
type ExecutionResultPolicy func(ctx context.Context, err error) limit.Result

// everyExternalErrorAsFailurePolicy will treat as failure every error that is not
// from concurrencylimit package (this is the error by the limiters).
var everyExternalErrorAsFailurePolicy = func(_ context.Context, err error) limit.Result {
	// Everything ok.
	if err == nil {
		return limit.ResultSuccess
	}

	// Our own failures should be ignored, the rest nope.
	if err != nil && err != errors.ErrRejectedExecution {
		return limit.ResultFailure
	}

	return limit.ResultIgnore
}

// Config is the concurrency limit algorithm
type Config struct {
	// Limiter is the algorithm implementation that will calculate
	// the limits in adaptive way.
	Limiter limit.Limiter
	// Executor is the implementation used to execute the functions internally. It maintains
	// the workers dynamically based on the CongestionControlAlgorithm limits.
	Executor execute.Executor
	// ExecutionResultPolicy is a function where the execution error will be passed along with
	// the context and return if that result should be treated as a success, an error or ignored
	// by the concurrency control algorithm.
	// By default every error will count as an error.
	ExecutionResultPolicy ExecutionResultPolicy
}

func (c *Config) defaults() {
	if c.Limiter == nil {
		c.Limiter = limit.NewStatic(10)
	}

	if c.Executor == nil {
		c.Executor = execute.NewBlocker(execute.BlockerConfig{})
	}

	if c.ExecutionResultPolicy == nil {
		c.ExecutionResultPolicy = everyExternalErrorAsFailurePolicy
	}
}

// New returns a new goresilience concurrency limit Runner.
func New(cfg Config) goresilience.Runner {
	return NewMiddleware(cfg)(nil)
}

// NewMiddleware returns a new concurrenct limit middleware
func NewMiddleware(cfg Config) goresilience.Middleware {
	cfg.defaults()

	// Set initial limit.
	cfg.Executor.SetWorkerQuantity(cfg.Limiter.GetLimit())

	return func(next goresilience.Runner) goresilience.Runner {
		c := &concurrencylimit{
			runner: goresilience.SanitizeRunner(next),
			cfg:    cfg,
		}

		return c
	}
}

type concurrencylimit struct {
	runner    goresilience.Runner
	inflights atomicCounter
	cfg       Config
}

func (c *concurrencylimit) Run(ctx context.Context, f goresilience.Func) error {
	start := time.Now()

	metricsRecorder, _ := metrics.RecorderFromContext(ctx)

	// Submit the job
	currentInflights := c.inflights.Inc()
	metricsRecorder.SetConcurrencyLimitInflightExecutions(currentInflights)

	err := c.cfg.Executor.Execute(func() error {
		return c.runner.Run(ctx, f)
	})

	currentInflights = c.inflights.Dec()
	metricsRecorder.SetConcurrencyLimitInflightExecutions(currentInflights)

	// Measure to feed the algorithm.
	result := c.cfg.ExecutionResultPolicy(ctx, err)
	metricsRecorder.IncConcurrencyLimitResult(string(result))

	limit := c.cfg.Limiter.MeasureSample(start, currentInflights, result)
	metricsRecorder.SetConcurrencyLimitLimiterLimit(limit)

	// Update the congestion window based on the new algorithm results.
	c.cfg.Executor.SetWorkerQuantity(limit)

	return err
}

type atomicCounter struct {
	c  int
	mu sync.Mutex
}

func (a *atomicCounter) Inc() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.c++
	return a.c
}

func (a *atomicCounter) Dec() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.c--
	return a.c
}
