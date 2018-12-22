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
	Pool
}

// Pool maintains a worker pool what knows how to increase and decrease the worker pool.
type Pool interface {
	SetWorkerQuantity(quantity int)
}

// pool knows how to increase and decrease the current workers executing jobs.
// it's only objective is to set the desired number of concurrent execution flows
type pool struct {
	workerStoppers []chan struct{}
	jobQueue       chan func()
	mu             sync.Mutex
}

func newPool() pool {
	return pool{
		jobQueue: make(chan func()),
	}
}

// SetWorkerQuantity knows how to increase or decrease the worker pool.
func (p *pool) SetWorkerQuantity(quantity int) {
	if quantity < 0 {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// If we don't need to increase or decrease the worker quantity the do nothing.
	if len(p.workerStoppers) == quantity {
		return
	}

	// If we have less workers then we need to add workers.
	if len(p.workerStoppers) < quantity {
		p.increaseWorkers(quantity - len(p.workerStoppers))
		return
	}

	// If we reached here then we need to reduce workers.
	p.decreaseWorkers(len(p.workerStoppers) - quantity)
}

func (p *pool) decreaseWorkers(workers int) {
	// Stop the not needed workers.
	toStop := p.workerStoppers[:workers]
	for _, stopC := range toStop {
		close(stopC)
	}

	// Set the new worker quantity.
	p.workerStoppers = p.workerStoppers[workers:]
}

func (p *pool) increaseWorkers(workers int) {
	for i := 0; i < workers; i++ {
		// Create a channel to stop the worker.
		stopC := make(chan struct{})
		go p.newWorker(stopC)
		p.workerStoppers = append(p.workerStoppers, stopC)
	}
}

func (p *pool) newWorker(stopC chan struct{}) {
	for {
		select {
		case <-stopC:
			return
		case f := <-p.jobQueue:
			f()
		}
	}
}
