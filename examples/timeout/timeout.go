package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/slok/goresilience/retry"
)

func main() {
	// Create our execution chain (nil marks the end of the chain).
	runner := retry.New(retry.Config{})

	for i := 0; i < 200; i++ {
		// Execute.
		result := ""
		err := runner.Run(context.TODO(), func(_ context.Context) error {
			if time.Now().Nanosecond()%2 == 0 {
				return errors.New("you didn't expect this error")
			}
			result = "all ok"
			return nil
		})

		if err != nil {
			result = "not ok, but fallback"
		}

		log.Printf("the result is: %s", result)
	}
}
