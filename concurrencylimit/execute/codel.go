package execute

import (
	"time"

	"github.com/slok/goresilience/errors"
)

// AdaptiveLIFOCodelConfig is the configuration for the AdaptiveLIFOCodel executor.
type AdaptiveLIFOCodelConfig struct {
	// CodelTarget is the duration the execution funcs can be on the queue before being
	// rejected when the controlled delay congestion has been activated (a.k.a bufferbloat detected).
	CodelTarget time.Duration
	// CodelInterval is the default max time the funcs can be on the queue before being rejected.
	CodelInterval time.Duration
	// The queue uses a goroutine in background to execute the queue
	// jobs, in case it want's to be stopped a channel could be used to
	// stop the execution.
	StopChannel chan struct{}
}

func (c *AdaptiveLIFOCodelConfig) defaults() {
	if c.StopChannel == nil {
		c.StopChannel = make(chan struct{})
	}

	if c.CodelTarget == 0 {
		c.CodelTarget = 5 * time.Millisecond
	}

	if c.CodelInterval == 0 {
		c.CodelInterval = 100 * time.Millisecond
	}
}

type adaptiveLIFOCodel struct {
	cfg   AdaptiveLIFOCodelConfig
	queue *queue
	workerPool
}

// NewAdaptiveLIFOCodel is an executor based on Codel algorithm (Controlled delay) for the execution,
// more info here https://queue.acm.org/detail.cfm?id=2209336, and adaptive LIFO for the queue
// priority.
//
// Codel implementation it's based on Facebook's Codel usage for resiliency.
// More information can be found here: https://queue.acm.org/detail.cfm?id=2839461
//
// At first the queue priority will be FIFO, but when we detect bufferbloat it will change queue
// execution priority to LIFO.
// On the other hand the execution timeout will change based on the last time the queue was empty
// this will give us the ability to set a lesser timeout on the queued executions when the queue
// starts to grow.
func NewAdaptiveLIFOCodel(cfg AdaptiveLIFOCodelConfig) Executor {

	cfg.defaults()

	a := &adaptiveLIFOCodel{
		cfg:        cfg,
		queue:      newQueue(cfg.StopChannel, enqueueAtEndPolicy, fifoDequeuePolicy),
		workerPool: newWorkerPool(),
	}
	go a.fromQueueToWorkerPool()

	return a
}

func (a *adaptiveLIFOCodel) Execute(f func() error) error {
	var timeout time.Duration
	// If we are congested then we need to change de queuing policy to LIFO
	// and set the congestion timeout to the aggressive CoDel timeout.
	if a.queueCongested() {
		a.queue.SetDequeuePolicy(lifoDequeuePolicy)
		timeout = a.cfg.CodelTarget
	} else {
		// No congestion means fifo and regular timeout.
		a.queue.SetDequeuePolicy(fifoDequeuePolicy)
		timeout = a.cfg.CodelInterval
	}

	// This channel will receive a signal if the job is cancelled.
	cancelJob := make(chan struct{})

	// Create a job and listen if the job has been cancelled before
	// executing something that will not be used due to an upper layer
	// already cancelled (this layer).
	res := make(chan error)
	job := func() {
		select {
		case <-cancelJob:
			return
		default:
		}

		// Don't wait if the channel has been cancelled due to a timeout.
		select {
		case res <- f():
		default:
		}
	}

	// Enqueue the job.
	go func() {
		a.queue.InChannel() <- job
	}()

	// Wait until executed or timeout.
	select {
	case <-time.After(timeout):
		close(cancelJob)
		return errors.ErrRejectedExecution
	case result := <-res:
		return result
	}
}

// fromQueueToWorkerPool will get from the queue in a loop the jobs to be
// executed by the worker pool.
func (a *adaptiveLIFOCodel) fromQueueToWorkerPool() {
	for {
		select {
		case <-a.cfg.StopChannel:
			return
		case job := <-a.queue.OutChannel():
			// Send to execution worker.
			a.workerPool.jobQueue <- job
		}
	}
}

// queueCongested will calculate if the queue is congested based on CoDel algorithm.
func (a *adaptiveLIFOCodel) queueCongested() bool {
	return time.Since(a.queue.LastEmptyTime()) > a.cfg.CodelInterval
}
