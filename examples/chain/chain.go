package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/slok/goresilience"
	"github.com/slok/goresilience/bulkhead"
	"github.com/slok/goresilience/retry"
	"github.com/slok/goresilience/timeout"
)

func main() {
	// Create our execution chain.
	cmd := goresilience.RunnerChain(
		bulkhead.NewMiddleware(bulkhead.Config{}),
		retry.NewMiddleware(retry.Config{}),
		timeout.NewMiddleware(timeout.Config{}),
	)

	// Execute.
	calledCounter := 0
	result := ""
	err := cmd.Run(context.TODO(), func(_ context.Context) error {
		calledCounter++
		if calledCounter%2 == 0 {
			return errors.New("you didn't expect this error")
		}
		result = "all ok"
		return nil
	})

	if err != nil {
		result = "not ok, but fallback"
	}

	fmt.Printf("result: %s", result)
}
