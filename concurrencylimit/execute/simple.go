package execute

import (
	"time"

	"github.com/slok/goresilience/errors"
)

// SimpleConfig is the configuration for the Simple limiter.
type SimpleConfig struct {
	// MaxWaitTime is the max time a limiter will wait to execute before
	// being dropped it's execution and be rejected.
	MaxWaitTime time.Duration
}

func (c *SimpleConfig) defaults() {
	if c.MaxWaitTime == 0 {
		c.MaxWaitTime = 1 * time.Second
	}
}

// NewSimple returns a simple that will execute if there are workers available, if not it will get blocked
// and queued in a random priority queue until one worker is free or the timeout is reached, in this last
// case the execution will be treat as rejected.
func NewSimple(cfg SimpleConfig) Executor {
	cfg.defaults()

	return &simple{
		pool: newPool(),
		cfg:  cfg,
	}
}

type simple struct {
	cfg SimpleConfig
	pool
}

// Execute satisfies Limiter interface.
func (s *simple) Execute(f func() error) error {
	result := make(chan error)
	job := func() {
		result <- f()
	}

	select {
	case s.jobQueue <- job:
		return <-result
	case <-time.After(s.cfg.MaxWaitTime):
		return errors.ErrRejectedExecution
	}
}
