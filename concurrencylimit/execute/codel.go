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
	cfg AdaptiveLIFOCodelConfig
	// queue is the queue used to control how the jobs are sent to the worker pool
	// it knows the different queue priority policies (FIFO, LIFO...).
	queue *dynamicQueue
	// worker pool is the one that will execute the jobs.
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
		queue:      newDynamicQueue(cfg.StopChannel, enqueueAtEndPolicy, fifoDequeuePolicy),
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
	canceledJob := make(chan struct{}, 1)
	// This channel will receive a signal when the job has been dequeued
	// to be processed.
	dequeuedJob := make(chan struct{})

	// res is the result channel where this function will grab the result
	// of the submitted job to the queue.
	// In order to not block the executed func and leak goroutines we create
	// a buffered channel in case the executor gets to a condition where it
	// has finished before we reach the result waiting state.
	// This way the executor does not need to wait until the result gets received
	// and the receiver doesn't need to be there before the execution has finished.
	res := make(chan error, 1)

	// Create a job and check if the job has been cancelled already
	// in case we need to discard the execution of the client job.
	job := func() {
		// Send the signal the job has been dequeued.
		close(dequeuedJob)

		// Check if the job was already cancelled (a lightweight context.Done)
		select {
		case <-canceledJob:
			return
		default:
		}

		// Execute the function and send the result over the buffered channel
		// to avoid getting blocked.
		res <- f()
	}

	// Enqueue the job in the queue that knows how to submit jobs to the worker
	// pool afterwards.
	go func() {
		a.queue.InChannel() <- job
	}()

	// Wait until dequeued or timeout in queue waiting to be executed.
	select {
	case <-time.After(timeout):
		canceledJob <- struct{}{}
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
			a.workerPool.jobQueue <- job
		}
	}
}

// queueCongested will calculate if the queue is congested based on CoDel algorithm.
func (a *adaptiveLIFOCodel) queueCongested() bool {
	return a.queue.SinceLastEmpty() > a.cfg.CodelInterval
}
