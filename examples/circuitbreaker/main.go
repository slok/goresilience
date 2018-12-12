package main

import (
	"context"
	"fmt"
	"time"

	"github.com/slok/goresilience/circuitbreaker"
	"github.com/slok/goresilience/errors"
)

func errorOnOddMinute(ctx context.Context) error {
	minute := time.Now().Minute()
	if minute%2 != 0 {
		return fmt.Errorf("error because %d minute is odd", minute)
	}

	return nil
}

func main() {
	cb := circuitbreaker.New(circuitbreaker.Config{
		//ErrorPercentThresholdToOpen:        50,
		//MinimumRequestToOpen:               20,
		//SuccessfulRequiredOnHalfOpen:       1,
		//WaitDurationInOpenState:            5 * time.Second,
		//MetricsSlidingWindowBucketQuantity: 10,
		//MetricsBucketDuration:              1 * time.Second,
	}, nil)

	for {
		time.Sleep(75 * time.Millisecond)

		err := cb.Run(context.TODO(), errorOnOddMinute)
		if err != nil {
			if err == errors.ErrCircuitOpen {
				fmt.Println("[!] circuit open")
			} else {
				fmt.Printf("[-] execution error: %s\n", err)
			}
		} else {
			fmt.Printf("[+] good\n")
		}
	}
}
