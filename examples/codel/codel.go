package main

import (
	"context"
	"fmt"
	"time"

	"github.com/slok/goresilience"
	"github.com/slok/goresilience/concurrencylimit"
	"github.com/slok/goresilience/concurrencylimit/execute"
	"github.com/slok/goresilience/concurrencylimit/limit"
)

const (
	workers           = 7
	rate              = 4
	rateDuration      = 1 * time.Second
	burst             = 300
	burstRateDuration = 50 * time.Millisecond
	burstTime         = 20 * time.Second
)

func main() {
	runner := concurrencylimit.New(concurrencylimit.Config{
		Executor: execute.NewAdaptiveLIFOCodel(execute.AdaptiveLIFOCodelConfig{}),
		Limiter:  limit.NewStatic(workers),
	})

	fmt.Printf("CoDel runner ready with %d workers...======================================\n", workers)

	// Create a Regular rate of request.
	fmt.Printf("start receiving a execution rate of  %d/s...======================================\n", rate)
	go runRate(runner, rate, rateDuration, 0)

	time.Sleep(5 * time.Second)
	// Execute a burst that will make the CoDel start working.
	fmt.Printf("receiving a burst execution rate of %d/s for %s...======================================\n", burst, burstTime)
	runRate(runner, burst, burstRateDuration, burstTime)

	fmt.Printf("stop burst...\n")
	time.Sleep(2 * time.Minute)

}

// runRate rate will create a rate (N per second) of executions.
// if duration is different from 0 it will set a max duration for the rate.
func runRate(runner goresilience.Runner, rate int, rateDuration time.Duration, duration time.Duration) {

	rateTick := time.NewTicker(rateDuration)
	var timeout = make(<-chan time.Time)
	if duration != 0 {
		timeout = time.After(duration)
	}

	for {
		select {
		case <-rateTick.C:
			for i := 0; i < rate; i++ {
				go handler(runner, "executed (in %s)\n", "load shedded (in %s)\n")
			}
		case <-timeout:
			return
		}
	}
}

// handler is the bussiness logic to execute using the runner.
func handler(runner goresilience.Runner, okmsg, failmsg string) {
	start := time.Now()
	err := runner.Run(context.TODO(), func(_ context.Context) error {
		lat := 1 * time.Second
		jitter := time.Duration(time.Now().UnixNano()%300) * time.Millisecond
		if time.Now().UnixNano()%2 == 0 {
			lat += jitter
		} else {
			lat -= jitter
		}

		time.Sleep(lat)

		if okmsg != "" {
			fmt.Printf(okmsg, time.Since(start))
		}
		return nil
	})

	if err != nil {
		fmt.Printf(failmsg, time.Since(start))
	}
}
