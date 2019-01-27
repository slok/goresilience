package execute_test

import (
	"fmt"
	"sync"
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
			// WARNING: This test is extremely complex, this is because of the nature of checking
			// the algorithm execution priorities, cancelations... sorry :/
			name: "An execution that detects congestion using CoDel should activate target timeout and use LIFO priority for the queue.",
			cfg: execute.AdaptiveLIFOCodelConfig{
				CodelTargetDelay: 10 * time.Millisecond,
				CodelInterval:    50 * time.Millisecond,
			},
			exec: func(exec execute.Executor) []string {
				// This test will probe the CoDel algorithm works. With one worker everything will
				// be easier.
				// 1 - Enqueue 10 jobs with a execution delay of 100ms (CoDel interval 50ms).
				// 2 - The first jobs will be executed in order(0, 1...)
				// 3 - After the first delayed work, CoDel triggers congestion and starts executing in LIFO (9, 8...)
				// 4 - The LIFO execution jobs (9, 8, 7...) have a lesser timeout than the FIFO ones (0, 1, 2), 10ms
				// 5 - The last jobs of the queue (they have been enqueued with CoDel active) will be timeout (10ms).
				// 6 - Give time so CoDel empties the queue (wait 200ms).
				// 7 - Add another 10 items with a low delay of execution.
				// 8 - The execution is FIFO due to not be in congestion after emptying the queue and this new jobs
				//	   not triggering congestion neither (a.k.a self healed by CoDel + adaptive LIFO).

				exec.SetWorkerQuantity(1)

				// Execute the result grabber in background. This function will be
				// grabbing the results constantly until we receive a stop signal.
				// We use a mutex so when the test returns the result ensure the background
				// grabber is not adding more results and we can be sure that we grabbed
				// everything that we could grab.
				resultC := make(chan string)
				stopGrabC := make(chan struct{})
				results := []string{}
				var resultsmu sync.Mutex
				go func() {
					// Lock results until finished grabbing results.
					resultsmu.Lock()
					defer resultsmu.Unlock()
					for {
						select {
						case <-stopGrabC:
							return
						case res := <-resultC:
							results = append(results, res)
						}
					}
				}()

				for i := 0; i < 10; i++ {
					// Sleep so the funcs enter in order and are executed by the
					// unique worker.
					time.Sleep(10 * time.Millisecond)
					i := i
					go func() {
						exec.Execute(func() error {
							time.Sleep(100 * time.Millisecond)
							resultC <- fmt.Sprintf("id-%d", i)
							return nil
						})
					}()
				}

				// Wait to empty the queue.
				time.Sleep(200 * time.Millisecond)

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

				// Wait some time to grab the inflight background executions.
				// then close the grabber.
				time.Sleep(200 * time.Millisecond)
				close(stopGrabC)

				// This lock will wait until the grabber unlocks the results
				resultsmu.Lock()
				defer resultsmu.Unlock()
				return results
			},
			expResults: []string{
				"id-0", // congestion activated.
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
