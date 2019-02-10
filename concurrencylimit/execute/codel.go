package execute

import (
	"time"

	"github.com/slok/goresilience/errors"
)

// AdaptiveLIFOCodelConfig is the configuration for the AdaptiveLIFOCodel executor.
type AdaptiveLIFOCodelConfig struct {
	// CodelTargetDelay is the duration the execution funcs can be on the queue before being
	// rejected when the controlled delay congestion has been activated (a.k.a bufferbloat detected).
	CodelTargetDelay time.Duration
	// CodelInterval is the default max time the funcs can be on the queue before being rejected.
	CodelInterval time.Duration
	// The queue uses a goroutine in background to execute the queue
	// jobs, in case it wants to be stopped a channel could be used to
	// stop the execution.
	StopChannel chan struct{}
}

func (c *AdaptiveLIFOCodelConfig) defaults() {
	if c.StopChannel == nil {
		c.StopChannel = make(chan struct{})
	}

	if c.CodelTargetDelay == 0 {
		c.CodelTargetDelay = 5 * time.Millisecond
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

// NewAdaptiveLIFOCodel is an executor based on CoDel algorithm (Controlled delay) for the execution,
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
		timeout = a.cfg.CodelTargetDelay
	} else {
		// No congestion means fifo and regular timeout.
		a.queue.SetDequeuePolicy(fifoDequeuePolicy)
		timeout = a.cfg.CodelInterval
	}

	// This channel will receive a signal if the job is cancelled.
	canceledJob := make(chan struct{})
	// This channel will receive a signal when the job has been dequeued
	// to be processed.
	dequeuedJob := make(chan struct{})

	// Create a job and listen if the job has been cancelled already.
	res := make(chan error)
	job := func() {
		// Send the signal the job has been dequeued.
		close(dequeuedJob)

		// Check if the job was already cancelled (a lightweight context.Done)
		select {
		case <-canceledJob:
			return
		default:
		}

		// Execute the function and don't wait if nobody is listening.
		// in the worst case we have done work for nothing but we don't
		// get blocked.
		err := f()
		select {
		case res <- err:
		default:
		}
	}

	// Enqueue the job.
	go func() {
		a.queue.InChannel() <- job
	}()

	// Wait until dequeued or timeout in queue waiting to be executed.
	select {
	case <-time.After(timeout):
		close(canceledJob)
		return errors.ErrRejectedExecution
	case <-dequeuedJob:
		return <-res
	}
}

// fromQueueToWorkerPool will get jobs from the queue in a loop and send
// to the worker pools to be executed.
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
