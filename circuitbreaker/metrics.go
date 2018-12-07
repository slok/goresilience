package circuitbreaker

import "sync"

// recorder knows how to record the request and errors for a circuitbreaker.
type recorder interface {
	inc(err error)
	reset()
	errorRate() float64
	totalRequests() float64
}

// counter records the metrics on a global counter.
type counter struct {
	total float64
	errs  float64
	mu    sync.Mutex
}

func (c *counter) inc(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.total++
	if err != nil {
		c.errs++
	}
}

func (c *counter) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.total = 0
	c.errs = 0
}

func (c *counter) errorRate() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.errs / c.total
}

func (c *counter) totalRequests() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.total
}
