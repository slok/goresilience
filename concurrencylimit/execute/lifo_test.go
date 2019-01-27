package execute_test

import (
	"testing"
	"time"

	"github.com/slok/goresilience/concurrencylimit/execute"
	"github.com/stretchr/testify/assert"
)

func TestExecuteLIFO(t *testing.T) {
	tests := []struct {
		name          string
		cfg           execute.LIFOConfig
		f             func() error
		numberCalls   int
		numberWorkers int
		expOK         int
	}{
		{
			name:          "A LIFO executor with a not aggresive timeout and sufficent workers should execute all.",
			cfg:           execute.LIFOConfig{},
			f:             func() error { return nil },
			numberCalls:   50,
			numberWorkers: 100,
			expOK:         50,
		},
		{
			name: "A simple executor with a an aggresive timeout and not sufficent workers should fail fast.",
			cfg: execute.LIFOConfig{
				MaxWaitTime: 5 * time.Millisecond,
			},
			f: func() error {
				time.Sleep(30 * time.Millisecond)
				return nil
			},
			numberCalls:   50,
			numberWorkers: 25,
			expOK:         25,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			exec := execute.NewLIFO(test.cfg)

			// Set the number of workers.
			exec.SetWorkerQuantity(test.numberWorkers)

			// Execute multiple concurrent calls.
			results := make(chan error)
			for i := 0; i < test.numberCalls; i++ {
				go func() {
					results <- exec.Execute(test.f)
				}()
			}

			// Grab the results.
			gotOK := 0
			for i := 0; i < test.numberCalls; i++ {
				if res := <-results; res == nil {
					gotOK++
				}
			}

			// Check the results.
			assert.Equal(test.expOK, gotOK)
		})
	}
}

func TestExecuteLIFOOrder(t *testing.T) {
	tests := []struct {
		name        string
		numberCalls int
		expResult   []int
	}{
		{
			name:        "In a LIFO queue the queued objects should execute in a first-in-first-out priority.",
			numberCalls: 12,
			// Due how the wait signals from the queue is implemented the first two
			// elements in the queue will be get before the others.
			expResult: []int{0, 1, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			exec := execute.NewLIFO(execute.LIFOConfig{
				MaxWaitTime: 500 * time.Second, // Long enough so doesn't timeout anything.
			})

			// Execute multiple concurrent calls.
			results := make(chan int)
			for i := 0; i < test.numberCalls; i++ {
				time.Sleep(1 * time.Millisecond)
				i := i
				go func() {
					exec.Execute(func() error {
						results <- i
						return nil
					})
				}()
			}
			time.Sleep(10 * time.Millisecond)

			// Set the number of workers. This will start draining the queue.
			exec.SetWorkerQuantity(1)

			// Grab the results.
			gotResult := []int{}
			for i := 0; i < test.numberCalls; i++ {
				gotResult = append(gotResult, <-results)
			}

			// Check the results.
			assert.Equal(test.expResult, gotResult)
		})
	}
}
