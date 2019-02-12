package execute

import (
	"sync"
	"time"
)

// dequeuePolicy will receive a queue of jobs and return a job and the result of the
// queue after dequeing the job.
type dequeuePolicy func(beforeJobQ []func()) (job func(), afterJobQ []func())

// enqueuePolicy will receive a queue of jobs and a job and will queue the job.
type enqueuePolicy func(job func(), beforeJobQ []func()) (afterJobQ []func())

// dynamicQueue is a queue that knows how to queue and dequeue objects using different kind of policies.
// these policies can be changed with the queue is running.
type dynamicQueue struct {
	in            chan func()
	out           chan func()
	policyMu      sync.RWMutex
	jobsMu        sync.Mutex
	jobs          []func()
	enqueuePolicy enqueuePolicy
	dequeuePolicy dequeuePolicy
	queueStats
	stopC chan struct{}
	// wakeupDequeuerC will be use to  wake up the dequeuer that has been sleeping due to no jobs on the queue.
	wakeUpDequeuerC chan struct{}
}

func newDynamicQueue(stopC chan struct{}, enqueuePolicy enqueuePolicy, dequeuePolicy dequeuePolicy) *dynamicQueue {
	q := &dynamicQueue{
		in:            make(chan func()),
		out:           make(chan func()),
		enqueuePolicy: enqueuePolicy,
		dequeuePolicy: dequeuePolicy,
		stopC:         stopC,
		// wakeUpDequeuerC will be used to wake up the dequeuer when the queue goes empty so we don't need
		// to poll the queue every T interval (is an optimization), this way the enqueuer will notify through
		// this channel the dequeuer that elements have been added and needs to wake up to dequeue those
		// new elements.
		//
		// We use a buffered channel so the queue doesn't get blocked/stuck forever, because it could happen that
		// the signal is sent when the dequeuer isn't listening and when it starts waiting, the signal has
		// been ignored. This is because the enqueuer doesn't get blocked when sending the signals to the dequeuer
		// through this channel, it notifies only if the dequeuer is listening. Using a buffered channel of 1 is enough
		// to tell the dequeuer that at least one job has been enqueued and it can wake up although it wasn't listening
		// at the time of notifying in the enqueue moment.
		// A drawback is that could happen that the dequeuer gets the buffered signal of an old and already queued element
		// and in the moment of waking up, the queue is empty, so that's why we need to check again if the queue is empty
		// just after waiking up the dequeuer.
		wakeUpDequeuerC: make(chan struct{}, 1),
	}

	// Start the background jobs that accept/return In/Out jobs.
	go q.dequeuer()
	go q.enqueuer()

	return q
}

// InChannel returns a channel where the queue will receive the jobs.
func (d *dynamicQueue) InChannel() chan<- func() {
	return d.in
}

// OutChannel returns a channel where the jobs of the queue can be dequeued.
func (d *dynamicQueue) OutChannel() <-chan func() {
	return d.out
}

func (d *dynamicQueue) SetEnqueuePolicy(p enqueuePolicy) {
	d.policyMu.Lock()
	defer d.policyMu.Unlock()
	d.enqueuePolicy = p
}

func (d *dynamicQueue) SetDequeuePolicy(p dequeuePolicy) {
	d.policyMu.Lock()
	defer d.policyMu.Unlock()
	d.dequeuePolicy = p
}

func (d *dynamicQueue) enqueuer() {
	for {
		select {
		case <-d.stopC:
			return
		case job := <-d.in:
			d.queueStats.inc() // Increase in 1 the queue stats.
			d.jobsMu.Lock()
			d.policyMu.RLock()
			d.jobs = d.enqueuePolicy(job, d.jobs)
			d.policyMu.RUnlock()
			// If the dequeuer is sleeping it will get the wake up signal, if not
			// the channel will not be being read and the default case will be selected.
			select {
			case d.wakeUpDequeuerC <- struct{}{}:
			default:
			}
			d.jobsMu.Unlock()
		}
	}
}

var x = 0

func (d *dynamicQueue) dequeuer() {
	for {
		select {
		case <-d.stopC:
			return
		default:
		}
		// If there are no jobs, instead of polling, sleep the dequeuer until
		// a job enters the queue, our enqueuer will try to wake up us when any
		// job is queued.
		if d.queueIsEmpty() {
			<-d.wakeUpDequeuerC

			// Check again after unblocking because could be the buffered channel signal
			// of a queue object that we had already processed.
			if d.queueIsEmpty() {
				continue
			}
		}
		// Get a new job
		var job func()
		d.jobsMu.Lock()
		d.policyMu.RLock()
		job, d.jobs = d.dequeuePolicy(d.jobs)
		d.policyMu.RUnlock()
		d.jobsMu.Unlock()
		d.queueStats.decr() // Reduce in 1 the queue stats.

		// Send the correct job with the channel.
		d.out <- job
	}
}

func (d *dynamicQueue) queueIsEmpty() bool {
	d.jobsMu.Lock()
	defer d.jobsMu.Unlock()
	return len(d.jobs) < 1
}

// Queue Policies.
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

// fifoDequeuePolicy implements the policy for a FIFO priority, it will
// dequeue de first job in the queue.
var fifoDequeuePolicy = func(queue []func()) (job func(), afterQueue []func()) {
	switch len(queue) {
	case 0:
		return nil, []func(){}
	default:
		return queue[0], queue[1:]
	}
}

// queueStats will manage the stats of the  queue (current
// inflight in queue, last time the queue was empty...).
type queueStats struct {
	lastTimeEmpty time.Time
	size          int
	mu            sync.Mutex
}

func (q *queueStats) inc() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.size <= 0 {
		q.lastTimeEmpty = time.Now()
	}
	q.size++
}

func (q *queueStats) decr() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.size--
	if q.size <= 0 {
		q.lastTimeEmpty = time.Now()
	}
}

// sinceLastEmpty will return how long has been been the queue empty.
func (q *queueStats) SinceLastEmpty() time.Duration {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.size <= 0 {
		q.lastTimeEmpty = time.Now()
	}
	return time.Since(q.lastTimeEmpty)
}
