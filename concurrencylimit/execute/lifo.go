package execute

import (
	"sync"
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
	mu    sync.Mutex
	queue *queue
	workerPool
}

// NewLIFO implements a LIFO priority executor.
func NewLIFO(cfg LIFOConfig) Executor {
	cfg.defaults()

	l := &lifo{
		cfg:        cfg,
		queue:      newQueue(cfg.StopChannel, enqueueAtEndPolicy, lifoDequeuePolicy),
		workerPool: newWorkerPool(),
	}
	go l.fromQueueToWorkerPool()

	return l
}

func (l *lifo) Execute(f func() error) error {
	start := time.Now()
	res := make(chan error)
	job := func() {
		// Maybe in the time of execution we have been waiting too much,
		// in that case don't execute and return error.
		if time.Since(start) > l.cfg.MaxWaitTime {
			res <- errors.ErrRejectedExecution
			return
		}

		res <- f()
	}

	// Send to a queue.
	l.queue.In <- job

	return <-res
}

// fromQueueToWorkerPool will get from the queue in a loop the jobs to be
// executed by the worker pool.
func (l *lifo) fromQueueToWorkerPool() {
	for {
		select {
		case <-l.cfg.StopChannel:
			return
		case job := <-l.queue.Out:
			// Send to execution worker.
			l.workerPool.jobQueue <- job
		}
	}
}

// enqueueAtEndPolicy enqueues at the end of the queue.
var enqueueAtEndPolicy = func(job func(), jobqueue []func()) []func() {
	return append(jobqueue, job)
}

// lifoDequeuePolicy implements the policy for a LIFO priority, it will
// dequeue de latest job queue.
var lifoDequeuePolicy = func(queue []func()) (job func(), afterQueue []func()) {
	switch len(queue) {
	case 0:
		return nil, []func(){}
	case 1:
		return queue[0], []func(){}
	default:
		// LIFO order, get the last one on the queued.
		length := len(queue)
		return queue[length-1], queue[:length-1]
	}
}
