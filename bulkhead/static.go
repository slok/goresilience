package bulkhead

import (
	"context"
	"time"

	"github.com/slok/goresilience"
	"github.com/slok/goresilience/errors"
	runnerutils "github.com/slok/goresilience/internal/util/runner"
	"github.com/slok/goresilience/metrics"
)

// StaticConfig is the configuration of the Static Bulkhead runner.
type StaticConfig struct {
	// Workers is the number of workers in the execution pool.
	Workers int
	// MaxWaitTime is the max time a runner will wait to execute before
	// being dropped it's execution and return a timeout error.
	MaxWaitTime time.Duration
	// StopC is a channel to stop the workers if required usually used for a graceful stop flow.
	StopC chan (struct{})
}

func (s *StaticConfig) defaults() {
	if s.Workers <= 0 {
		s.Workers = 15
	}

	if s.MaxWaitTime < 0 {
		s.MaxWaitTime = 0
	}

	if s.StopC == nil {
		s.StopC = make(chan struct{})
	}
}

type staticBulkhead struct {
	cfg    StaticConfig
	runner goresilience.Runner
	jobC   chan func() // jobC is the channel used to send job to the worker pool.
}

// NewStatic returns a new buklhead static runner.
// Static bulkhead will limit the execution of execution blocks based on
// a static configuration. The bulkhead implementation will be made
// using a worker of pools, the workers will pick these execution blocks
// when they are free, they will execute the logic and pick another block
// the exeuction block will wait to be picked by the workers and if they
// have a max wait time, if that time is passed they will be dropped
// from the execution queue.
func NewStatic(cfg StaticConfig, r goresilience.Runner) goresilience.Runner {
	r = runnerutils.Sanitize(r)

	cfg.defaults()

	s := &staticBulkhead{
		cfg:    cfg,
		runner: r,
		jobC:   make(chan func()),
	}

	// Our workers in background.
	go s.startWorkerPool()

	return s
}

func (s staticBulkhead) Run(ctx context.Context, f goresilience.Func) error {
	metricsRecorder, _ := metrics.RecorderFromContext(ctx)

	resC := make(chan error) // The result channel.
	job := func() {
		metricsRecorder.IncBulkheadProcessed()
		resC <- s.runner.Run(ctx, f)
	}

	metricsRecorder.IncBulkheadQueued()
	if s.cfg.MaxWaitTime == 0 {
		select {
		// Send the function to the worker
		case s.jobC <- job:
			// Wait for the result on the result channel.
			return <-resC
		}
	} else {
		select {
		case <-time.After(s.cfg.MaxWaitTime):
			metricsRecorder.IncBulkheadTimeout()
			return errors.ErrTimeoutWaitingForExecution
		// Send the function to the worker
		case s.jobC <- job:
			// Wait for the result on the result channel.
			return <-resC
		}
	}
}

// startWorkerPool will start the execution of the worker pool.
func (s staticBulkhead) startWorkerPool() {
	for i := 0; i < s.cfg.Workers; i++ {
		// Run worker.
		go func() {
			for {
				select {
				case <-s.cfg.StopC:
					return
				case job := <-s.jobC:
					job()
				}
			}
		}()
	}
}
