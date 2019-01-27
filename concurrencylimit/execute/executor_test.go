package execute_test

import (
	"sync"
	"testing"

	"github.com/slok/goresilience/concurrencylimit/execute"
)

var benchf = func() error {
	c := 1
	for i := 1; i <= 5; i++ {
		c = c * i
	}
	return nil
}

func BenchmarkExecutors(b *testing.B) {
	b.StopTimer()

	benchs := []struct {
		name        string
		getExecutor func(stopC chan struct{}) execute.Executor
	}{
		{
			name: "Benchmark with FIFO (10 workers).",
			getExecutor: func(_ chan struct{}) execute.Executor {
				e := execute.NewFIFO(execute.FIFOConfig{})
				e.SetWorkerQuantity(10)
				return e
			},
		},
		{
			name: "Benchmark with LIFO (10 workers).",
			getExecutor: func(stopC chan struct{}) execute.Executor {
				e := execute.NewLIFO(execute.LIFOConfig{
					StopChannel: stopC,
				})
				e.SetWorkerQuantity(10)
				return e
			},
		},
		{
			name: "Benchmark with adaptive LIFO + CoDel (10 workers).",
			getExecutor: func(stopC chan struct{}) execute.Executor {
				e := execute.NewAdaptiveLIFOCodel(execute.AdaptiveLIFOCodelConfig{
					StopChannel: stopC,
				})
				e.SetWorkerQuantity(10)
				return e
			},
		},
	}

	for _, bench := range benchs {
		b.Run(bench.name, func(b *testing.B) {
			// Prepare.
			stopC := make(chan struct{})
			defer close(stopC)
			exec := bench.getExecutor(stopC)

			// Make the executions.
			for n := 0; n < b.N; n++ {
				b.StartTimer()

				var wg sync.WaitGroup
				wg.Add(50)
				for i := 0; i < 50; i++ {
					go func() {
						defer wg.Done()
						exec.Execute(benchf)
					}()
				}
				wg.Wait()

				b.StopTimer()
			}
		})
	}
}
