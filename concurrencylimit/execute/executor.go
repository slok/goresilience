package execute

import "sync"

// Executor knows how to limit the execution using different kind of execution workflows
// like worker pools.
// It also has different policies of how to work, for example waiting a time before
// erroring, or directly erroring.
type Executor interface {
	// Execute will execute the received function and will return  the
	// rsult of the executed funciton, or reject error from the executor.
	Execute(f func() error) error
	WorkerPool
}

// WorkerPool maintains a worker pool what knows how to increase and decrease the worker pool.
type WorkerPool interface {
	SetWorkerQuantity(quantity int)
}

// workerPool knows how to increase and decrease the current workers executing jobs.
// it's only objective is to set the desired number of concurrent execution flows
type workerPool struct {
	workerStoppers []chan struct{}
	jobQueue       chan func()
	mu             sync.Mutex
}

func newWorkerPool() workerPool {
	return workerPool{
		jobQueue: make(chan func()),
	}
}

// SetWorkerQuantity knows how to increase or decrease the worker pool.
func (w *workerPool) SetWorkerQuantity(quantity int) {
	if quantity < 0 {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// If we don't need to increase or decrease the worker quantity the do nothing.
	if len(w.workerStoppers) == quantity {
		return
	}

	// If we have less workers then we need to add workers.
	if len(w.workerStoppers) < quantity {
		w.increaseWorkers(quantity - len(w.workerStoppers))
		return
	}

	// If we reached here then we need to reduce workers.
	w.decreaseWorkers(len(w.workerStoppers) - quantity)
}

func (w *workerPool) decreaseWorkers(workers int) {
	// Stop the not needed workers.
	toStop := w.workerStoppers[:workers]
	for _, stopC := range toStop {
		close(stopC)
	}

	// Set the new worker quantity.
	w.workerStoppers = w.workerStoppers[workers:]
}

func (w *workerPool) increaseWorkers(workers int) {
	for i := 0; i < workers; i++ {
		// Create a channel to stop the worker.
		stopC := make(chan struct{})
		go w.newWorker(stopC)
		w.workerStoppers = append(w.workerStoppers, stopC)
	}
}

func (w *workerPool) newWorker(stopC chan struct{}) {
	for {
		select {
		case <-stopC:
			return
		case f := <-w.jobQueue:
			f()
		}
	}
}
