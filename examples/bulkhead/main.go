package main

import (
	"context"
	"fmt"
	"time"

	"github.com/slok/goresilience/bulkhead"
)

const (
	times = 500
)

func main() {
	runner := bulkhead.New(bulkhead.Config{
		Workers:     20,
		MaxWaitTime: 10 * time.Millisecond,
	})

	// Execute our logic at the same time.
	for i := 0; i < times; i++ {
		i := i

		// Every 10 cmd executions we wait.
		if i%10 == 0 {
			time.Sleep(20 * time.Millisecond)
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

	time.Sleep(10 * time.Second)
}
