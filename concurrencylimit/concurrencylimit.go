package concurrencylimit

import (
	"context"
	"sync"
	"time"

	"github.com/slok/goresilience"
	"github.com/slok/goresilience/concurrencylimit/execute"
	"github.com/slok/goresilience/concurrencylimit/limit"
	"github.com/slok/goresilience/metrics"
)

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
		c.Limiter = limit.NewAIMD(limit.AIMDConfig{})
	}

	if c.Executor == nil {
		c.Executor = execute.NewFIFO(execute.FIFOConfig{})
	}

	if c.ExecutionResultPolicy == nil {
		c.ExecutionResultPolicy = FailureOnRejectedPolicy
	}
}

// New returns a new goresilience concurrency limit Runner.
func New(cfg Config) goresilience.Runner {
	return NewMiddleware(cfg)(nil)
}

// NewMiddleware returns a new concurrent limit middleware
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
	executing atomicCounter
	cfg       Config
}

func (c *concurrencylimit) Run(ctx context.Context, f goresilience.Func) error {
	start := time.Now()

	metricsRecorder, _ := metrics.RecorderFromContext(ctx)

	// Submit the job.
	currentInflights := c.inflights.Inc()
	metricsRecorder.SetConcurrencyLimitInflightExecutions(currentInflights)

	var queuedDuration time.Duration // The time in queue.
	var executing int                // The current executing number of funcs.
	err := c.cfg.Executor.Execute(func() error {
		// At this point we are being executed, this means we have been dequeued.
		queuedDuration = time.Since(start)
		metricsRecorder.ObserveConcurrencyLimitQueuedTime(start)
		executing = c.executing.Inc()
		metricsRecorder.SetConcurrencyLimitExecutingExecutions(executing)
		defer func() {
			executing = c.executing.Dec()
			metricsRecorder.SetConcurrencyLimitExecutingExecutions(executing)
		}()

		// Execute the logic.
		return c.runner.Run(ctx, f)
	})

	currentInflights = c.inflights.Dec()
	metricsRecorder.SetConcurrencyLimitInflightExecutions(currentInflights)

	// Measure to feed the algorithm.
	result := c.cfg.ExecutionResultPolicy(ctx, err)
	metricsRecorder.IncConcurrencyLimitResult(string(result))
	// If the result is an ignore then we don't measure nor set a new limit.
	if result == limit.ResultIgnore {
		return err
	}

	limit := c.cfg.Limiter.MeasureSample(start, queuedDuration, currentInflights, result)
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
