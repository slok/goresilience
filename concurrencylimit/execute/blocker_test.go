package execute_test

import (
	"testing"
	"time"

	"github.com/slok/goresilience/concurrencylimit/execute"
	"github.com/stretchr/testify/assert"
)

var fOK = func() error { return nil }
var fSleep10ms = func() error {
	time.Sleep(10 * time.Millisecond)
	return nil
}

func TestExecuteBlocker(t *testing.T) {
	tests := []struct {
		name          string
		cfg           execute.BlockerConfig
		f             func() error
		numberCalls   int
		numberWorkers int
		expOK         int
	}{
		{
			name:          "A blocking executor with a not aggresive timeout and sufficent workers should execute all.",
			cfg:           execute.BlockerConfig{},
			f:             fOK,
			numberCalls:   50,
			numberWorkers: 100,
			expOK:         50,
		},
		{
			name: "A blocking executor with a an aggresive timeout and not sufficent workers should fail fast.",
			cfg: execute.BlockerConfig{
				MaxWaitTime: 2 * time.Millisecond,
			},
			f:             fSleep10ms,
			numberCalls:   50,
			numberWorkers: 25,
			expOK:         25,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			exec := execute.NewBlocker(test.cfg)

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
