package execute_test

import (
	"testing"
	"time"

	"github.com/slok/goresilience/concurrencylimit/execute"
	"github.com/stretchr/testify/assert"
)

func TestExecuteFIFO(t *testing.T) {
	tests := []struct {
		name          string
		cfg           execute.FIFOConfig
		f             func() error
		numberCalls   int
		numberWorkers int
		expOK         int
	}{
		{
			name:          "A FIFO executor with a not aggresive timeout and sufficent workers should execute all.",
			cfg:           execute.FIFOConfig{},
			f:             func() error { return nil },
			numberCalls:   50,
			numberWorkers: 100,
			expOK:         50,
		},
		{
			name: "A simple executor with a an aggresive timeout and not sufficent workers should fail fast.",
			cfg: execute.FIFOConfig{
				MaxWaitTime: 10 * time.Nanosecond,
			},
			f: func() error {
				time.Sleep(10 * time.Millisecond)
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

			exec := execute.NewFIFO(test.cfg)

			// Set the number of workers.
			exec.SetWorkerQuantity(test.numberWorkers)

			// Execute multiple concurrent cals.
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

func TestExecuteFIFOOrder(t *testing.T) {
	tests := []struct {
		name        string
		numberCalls int
		expResult   []int
	}{
		{
			name:        "In a FIFO queue the queued objects should execute in a first-in-first-out priority.",
			numberCalls: 12,
			expResult:   []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			exec := execute.NewFIFO(execute.FIFOConfig{
				MaxWaitTime: 500 * time.Millisecond, // Long enough so doesn't timeout anything.
			})

			// Set the number of workers.
			exec.SetWorkerQuantity(1)

			// Execute multiple concurrent calls.
			results := make(chan int)
			for i := 0; i < test.numberCalls; i++ {
				// sleep on each iteration to guarantee that the goroutines are executed
				// in order, but sleep less that the time the goroutine lasts executing to
				// queue executions.
				time.Sleep(1 * time.Millisecond)
				i := i
				go func() {
					exec.Execute(func() error {
						time.Sleep(2 * time.Millisecond)
						results <- i
						return nil
					})
				}()
			}

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
