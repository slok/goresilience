package bulkhead

import (
	"context"
	"time"

	"github.com/slok/goresilience"
	"github.com/slok/goresilience/errors"
	"github.com/slok/goresilience/metrics"
)

// Config is the configuration of the Bulkhead runner.
type Config struct {
	// Workers is the number of workers in the execution pool.
	Workers int
	// MaxWaitTime is the max time a runner will wait to execute before
	// being dropped it's execution and return a timeout error.
	MaxWaitTime time.Duration
	// StopC is a channel to stop the workers if required usually used for a graceful stop flow.
	StopC chan (struct{})
}

func (c *Config) defaults() {
	if c.Workers <= 0 {
		c.Workers = 15
	}

	if c.MaxWaitTime < 0 {
		c.MaxWaitTime = 0
	}

	if c.StopC == nil {
		c.StopC = make(chan struct{})
	}
}

type bulkhead struct {
	cfg    Config
	runner goresilience.Runner
	jobC   chan func() // jobC is the channel used to send job to the worker pool.
}

// New returns a new bulkhead runner.
// Bulkhead will limit the execution of execution blocks based on
// a configuration. The bulkhead implementation will be made
// using a worker of pools, the workers will pick these execution blocks
// when they are free, they will execute the logic and pick another block
// the execution block will wait to be picked by the workers and if they
// have a max wait time, if that time is passed they will be dropped
// from the execution queue.
func New(cfg Config) goresilience.Runner {
	return NewMiddleware(cfg)(nil)
}

// NewMiddleware returns a new middleware for the runner that returns
//  bulkhead.New.
func NewMiddleware(cfg Config) goresilience.Middleware {
	cfg.defaults()

	return func(next goresilience.Runner) goresilience.Runner {
		b := &bulkhead{
			cfg:    cfg,
			runner: goresilience.SanitizeRunner(next),
			jobC:   make(chan func()),
		}

		// Our workers in background.
		go b.startWorkerPool()

		return b
	}
}

func (b bulkhead) Run(ctx context.Context, f goresilience.Func) error {
	metricsRecorder, _ := metrics.RecorderFromContext(ctx)

	resC := make(chan error, 1) // The result channel.
	job := func() {
		metricsRecorder.IncBulkheadProcessed()
		resC <- b.runner.Run(ctx, f)
	}

	metricsRecorder.IncBulkheadQueued()
	if b.cfg.MaxWaitTime == 0 {
		select {
		// Send the function to the worker
		case b.jobC <- job:
			// Wait for the result on the result channel.
			return <-resC
		}
	} else {
		select {
		case <-time.After(b.cfg.MaxWaitTime):
			metricsRecorder.IncBulkheadTimeout()
			return errors.ErrTimeoutWaitingForExecution
		// Send the function to the worker
		case b.jobC <- job:
			// Wait for the result on the result channel.
			return <-resC
		}
	}
}

// startWorkerPool will start the execution of the worker pool.
func (b bulkhead) startWorkerPool() {
	for i := 0; i < b.cfg.Workers; i++ {
		// Run worker.
		go func() {
			for {
				select {
				case <-b.cfg.StopC:
					return
				case job := <-b.jobC:
					job()
				}
			}
		}()
	}
}
