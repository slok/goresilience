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

// queue is a queue that knows how to queue and dequeue objects using different kind of policies.
// these policies can be changed with the queue is running.
type queue struct {
	in            chan func()
	out           chan func()
	mu            sync.Mutex
	policyMu      sync.RWMutex
	jobs          []func()
	enqueuePolicy enqueuePolicy
	dequeuePolicy dequeuePolicy
	stopC         chan struct{}
	lastEmptyTime time.Time
	// wakeupDequeuerC will be use to  wake up the dequeuer that has been sleeping due to no jobs on the queue.
	wakeUpDequeuerC chan struct{}
}

func newQueue(stopC chan struct{}, enqueuePolicy enqueuePolicy, dequeuePolicy dequeuePolicy) *queue {
	q := &queue{
		in:            make(chan func()),
		out:           make(chan func()),
		enqueuePolicy: enqueuePolicy,
		dequeuePolicy: dequeuePolicy,
		stopC:         stopC,
		lastEmptyTime: time.Now(),
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

func (q *queue) LastEmptyTime() time.Time {
	// No need a mutex, assume small differences that will end
	// in the same state in favor of better performance.
	if q.queueIsEmpty() {
		q.lastEmptyTime = time.Now()
	}
	return q.lastEmptyTime
}

// InChannel returns a channel where the queue will receive the jobs.
func (q *queue) InChannel() chan<- func() {
	return q.in
}

// OutChannel returns a channel where the jobs of the queue can be dequeued.
func (q *queue) OutChannel() <-chan func() {
	return q.out
}

func (q *queue) SetEnqueuePolicy(p enqueuePolicy) {
	q.policyMu.Lock()
	defer q.policyMu.Unlock()
	q.enqueuePolicy = p
}

func (q *queue) SetDequeuePolicy(p dequeuePolicy) {
	q.policyMu.Lock()
	defer q.policyMu.Unlock()
	q.dequeuePolicy = p
}

func (q *queue) enqueuer() {
	for {
		select {
		case <-q.stopC:
			return
		case job := <-q.in:
			q.mu.Lock()
			q.policyMu.RLock()
			q.jobs = q.enqueuePolicy(job, q.jobs)
			q.policyMu.RUnlock()
			// If the dequeuer is sleeping it will get the wake up signal, if not
			// the channel will not be being read and the default case will be selected.
			select {
			case q.wakeUpDequeuerC <- struct{}{}:
			default:
			}
			q.mu.Unlock()
		}
	}
}

var x = 0

func (q *queue) dequeuer() {
	for {
		select {
		case <-q.stopC:
			return
		default:
		}
		// If there are no jobs, instead of polling, sleep the dequeuer until
		// a job enters the queue, our enqueuer will try to wake up us when any
		// job is queued.
		if q.queueIsEmpty() {
			<-q.wakeUpDequeuerC

			// Check again after unblocking because could be the buffered channel signal
			// of a queue object that we had already processed.
			if q.queueIsEmpty() {
				continue
			}
		}
		var job func()
		q.mu.Lock()
		q.policyMu.RLock()
		job, q.jobs = q.dequeuePolicy(q.jobs)
		q.policyMu.RUnlock()
		q.mu.Unlock()
		// Send the correct job with the channel.
		q.out <- job
	}
}

func (q *queue) queueIsEmpty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.jobs) < 1
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
