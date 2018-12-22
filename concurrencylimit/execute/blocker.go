package execute

import (
	"time"

	"github.com/slok/goresilience/errors"
)

// BlockerConfig is the configuration for the Blocker limiter.
type BlockerConfig struct {
	// MaxWaitTime is the max time a limiter will wait to execute before
	// being dropped it's execution and be rejected.
	MaxWaitTime time.Duration
}

func (c *BlockerConfig) defaults() {
	if c.MaxWaitTime == 0 {
		c.MaxWaitTime = 1 * time.Second
	}
}

// NewBlocker returns a blocker that will execute if there are workers available, if not it will get blocked
// until one worker is free or the timeout is reached, in this case the execution will be treat as rejected.
func NewBlocker(cfg BlockerConfig) Executor {
	cfg.defaults()

	return &blocker{
		pool: newPool(),
		cfg:  cfg,
	}
}

type blocker struct {
	cfg BlockerConfig
	pool
}

// Execute satisfies Limiter interface.
func (b *blocker) Execute(f func() error) error {
	result := make(chan error)
	job := func() {
		result <- f()
	}

	select {
	case b.jobQueue <- job:
		return <-result
	case <-time.After(b.cfg.MaxWaitTime):
		return errors.ErrRejectedExecution
	}
}
