package main

import (
	"context"
	"fmt"
	"time"

	"github.com/slok/goresilience/concurrencylimit"
	"github.com/slok/goresilience/concurrencylimit/limit"
)

const times = 100000

func main() {
	runner := concurrencylimit.New(concurrencylimit.Config{
		Limiter: limit.NewAIMD(limit.AIMDConfig{}),
	})

	// Execute our logic at the same time.
	for i := 0; i < times; i++ {
		i := i

		// Every 10 cmd executions we wait.
		if i%10 == 0 {
			time.Sleep(1 * time.Millisecond)
		}

		// Submit our command.
		go func() {
			err := runner.Run(context.TODO(), func(_ context.Context) error {
				time.Sleep(50 * time.Millisecond)
				fmt.Printf("[%d] executed\n", i)
				return nil
			})

			if err != nil {
				fmt.Printf("[%d] error: %s\n", i, err)
			}
		}()
	}

	time.Sleep(5 * time.Second)
}
