package execute

import (
	"time"

	"github.com/slok/goresilience/errors"
)

// LIFOConfig is the configuration for the LIFO executor.
type LIFOConfig struct {
	// MaxWaitTime is the max time a limiter will wait to execute before
	// being dropped it's execution and be rejected.
	MaxWaitTime time.Duration

	// The fifo queue uses a goroutine in background to execute the queue
	// jobs, in case it want's to be stopped a channel could be used to
	// stop the execution.
	StopChannel chan struct{}
}

func (c *LIFOConfig) defaults() {
	if c.MaxWaitTime == 0 {
		c.MaxWaitTime = 1 * time.Second
	}

	if c.StopChannel == nil {
		c.StopChannel = make(chan struct{})
	}
}

type lifo struct {
	cfg   LIFOConfig
	queue *dynamicQueue
	workerPool
}

// NewLIFO implements a LIFO priority executor.
func NewLIFO(cfg LIFOConfig) Executor {
	cfg.defaults()

	l := &lifo{
		cfg:        cfg,
		queue:      newDynamicQueue(cfg.StopChannel, enqueueAtEndPolicy, lifoDequeuePolicy),
		workerPool: newWorkerPool(),
	}
	go l.fromQueueToWorkerPool()

	return l
}

func (l *lifo) Execute(f func() error) error {
	// This channel will receive a signal when the job has been dequeued
	// to be processed.
	dequeuedJob := make(chan struct{})
	canceledJob := make(chan struct{})
	res := make(chan error)
	job := func() {
		// Send the signal the job has been dequeued.
		close(dequeuedJob)

		select {
		case <-canceledJob:
			return
		default:
		}

		res <- f()
	}

	// Send to a queue.
	go func() {
		l.queue.InChannel() <- job
	}()

	select {
	case <-time.After(l.cfg.MaxWaitTime):
		close(canceledJob)
		return errors.ErrRejectedExecution
	case <-dequeuedJob:
		return <-res
	}
}

// fromQueueToWorkerPool will get from the queue in a loop the jobs to be
// executed by the worker pool.
func (l *lifo) fromQueueToWorkerPool() {
	for {
		select {
		case <-l.cfg.StopChannel:
			return
		case job := <-l.queue.OutChannel():
			// Send to execution worker.
			l.workerPool.jobQueue <- job
		}
	}
}
