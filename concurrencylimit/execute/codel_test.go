package execute_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/slok/goresilience/concurrencylimit/execute"
	"github.com/stretchr/testify/assert"
)

func TestCodel(t *testing.T) {
	tests := []struct {
		name       string
		cfg        execute.AdaptiveLIFOCodelConfig
		exec       func(exec execute.Executor) (results []string)
		expResults []string
	}{
		{
			name: "A regular execution using CoDel should not activate target timeout and use FIFO priority for the queue.",
			cfg:  execute.AdaptiveLIFOCodelConfig{},
			exec: func(exec execute.Executor) []string {
				exec.SetWorkerQuantity(1)

				resultC := make(chan string)
				for i := 0; i < 10; i++ {
					// Sleep so the funcs enter in order and are executed by the
					// unique worker.
					time.Sleep(2 * time.Millisecond)
					i := i
					go func() {
						exec.Execute(func() error {
							time.Sleep(1 * time.Millisecond)
							resultC <- fmt.Sprintf("id-%d", i)
							return nil
						})
					}()
				}

				// Grab results.
				results := []string{}
				for i := 0; i < 10; i++ {
					res := <-resultC
					results = append(results, res)
				}

				return results
			},
			expResults: []string{"id-0", "id-1", "id-2", "id-3", "id-4", "id-5", "id-6", "id-7", "id-8", "id-9"},
		},
		{
			name: "An execution that detects congestion using CoDel should activate target timeout and use LIFO priority for the queue.",
			cfg: execute.AdaptiveLIFOCodelConfig{
				CodelTarget:   2 * time.Millisecond,
				CodelInterval: 5 * time.Millisecond,
			},
			exec: func(exec execute.Executor) []string {
				// This test will probe the CoDel algorithm works. With one worker everything will
				// be easier.
				// 1 - Enqueue 10 jobs with a execution delay of 5m (CoDel interval 5ms).
				// 2 - The first jobs will be executed in order(0, 1...)
				// 3 - After the first delayed work, CoDel triggers congestion and starts executing in LIFO (9, 8...)
				// 4 - The LIFO execution jobs (9, 8, 7...) have a lesser timeout than the FIFO ones (0, 1, 2)
				// 5 - The last jobs of the queue (they have been enqueued with CoDel active) will be timeout.
				// 6 - give time so CoDel empties the queue.
				// 7 - Add another 10 items with a low delay of execution.
				// 8 - The execution is FIFO due to not be in congestion after emptying the queue and this new jobs
				//	   not triggering congestion neither.

				exec.SetWorkerQuantity(1)

				resultC := make(chan string)

				// Execute the result grab.
				results := []string{}
				go func() {
					for res := range resultC {
						results = append(results, res)
					}
				}()

				for i := 0; i < 10; i++ {
					// Sleep so the funcs enter in order and are executed by the
					// unique worker.
					time.Sleep(1 * time.Millisecond)
					i := i
					go func() {
						exec.Execute(func() error {
							time.Sleep(5 * time.Millisecond)
							resultC <- fmt.Sprintf("id-%d", i)
							return nil
						})
					}()
				}

				// Wait to empty the queue.
				time.Sleep(100 * time.Millisecond)

				for i := 10; i < 15; i++ {
					// Sleep so the funcs enter in order and are executed by the
					// unique worker.
					time.Sleep(1 * time.Millisecond)
					i := i
					go func() {
						exec.Execute(func() error {
							time.Sleep(1 * time.Millisecond)
							resultC <- fmt.Sprintf("id-%d", i)
							return nil
						})
					}()
				}

				// Wait for final results grab.
				time.Sleep(100 * time.Millisecond)

				return results
			},
			expResults: []string{
				"id-0", "id-1", // congestion activated.
				"id-9", // enqueued in congestion, dequeued in FIFO.
				// Timeout all the ones when enqueued with congestion (2, 3, 4, 5, 6, 7, 8)
				"id-10", "id-11", "id-12", "id-13", "id-14", // Executed in LIFO (without congestion).
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			// Prepare executor.
			codel := execute.NewAdaptiveLIFOCodel(test.cfg)

			// Execute.
			gotResults := test.exec(codel)

			// Check.
			assert.Equal(test.expResults, gotResults)
		})
	}
}
