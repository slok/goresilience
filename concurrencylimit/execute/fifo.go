package execute

import (
	"context"
	"time"

	"github.com/slok/goresilience/errors"
)

// FIFOConfig is the configuration for the FIFO limiter.
type FIFOConfig struct {
	// MaxWaitTime is the max time a limiter will wait to execute before
	// being dropped it's execution and be rejected.
	MaxWaitTime time.Duration
}

func (c *FIFOConfig) defaults() {
	if c.MaxWaitTime == 0 {
		c.MaxWaitTime = 1 * time.Second
	}
}

// NewFIFO returns a FIFO executor that will execute if there are workers available, if not it will get blocked
// and queued with FIFO priority until one worker is free or the timeout is reached, in this last
// case the execution will be treat as rejected.
//
// The FIFO kind queue is based on internal implementation of Go channels that makes blocked sends to a
// channel execute in a first-in-first-out priority.
func NewFIFO(cfg FIFOConfig) Executor {
	cfg.defaults()

	return &fifo{
		workerPool: newWorkerPool(),
		cfg:        cfg,
	}
}

type fifo struct {
	cfg FIFOConfig
	workerPool
}

// Execute satisfies Executor interface.
func (f *fifo) Execute(_ context.Context, fn func() error) error {
	result := make(chan error)
	job := func() {
		result <- fn()
	}

	select {
	case f.jobQueue <- job:
		return <-result
	case <-time.After(f.cfg.MaxWaitTime):
		return errors.ErrRejectedExecution
	}
}
