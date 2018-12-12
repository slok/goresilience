package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/slok/goresilience"
	"github.com/slok/goresilience/bulkhead"
	"github.com/slok/goresilience/circuitbreaker"
	"github.com/slok/goresilience/metrics"
	"github.com/slok/goresilience/retry"
	"github.com/slok/goresilience/timeout"
)

// HystrixConf is the configuration of the hystrix runner.
type HystrixConf struct {
	Circuitbreaker  circuitbreaker.Config
	Bulkhead        bulkhead.Config
	Retry           retry.Config
	Timeout         timeout.Config
	ID              string
	MetricsRecorder metrics.Recorder
}

// NewHystrix returns a runner that simulates the most used features
// of Netflix Hystrix library.
//
// It is a circuit breaker with a bulkhead, with a retry with timeout,
// in that order.
func NewHystrix(cfg HystrixConf) goresilience.Runner {
	// The order of creating a Hystrix runner is:
	// circuitbreaker -> bulkhead -> retry -> timeout
	hystrixRunner := circuitbreaker.New(cfg.Circuitbreaker,
		bulkhead.New(cfg.Bulkhead,
			retry.New(cfg.Retry,
				timeout.New(cfg.Timeout, nil))))

	return metrics.NewMeasuredRunner(cfg.ID, cfg.MetricsRecorder, hystrixRunner)
}

type result struct {
	err error
	msg string
}

func main() {
	// Prometheus registry to expose metrics.
	promreg := prometheus.NewRegistry()
	go func() {
		http.ListenAndServe(":8081", promhttp.HandlerFor(promreg, promhttp.HandlerOpts{}))
	}()

	runner := NewHystrix(HystrixConf{
		ID:              "hystrix-example",
		MetricsRecorder: metrics.NewPrometheusRecorder(promreg),
		Bulkhead: bulkhead.Config{
			MaxWaitTime: 6 * time.Second,
		},
	})
	results := make(chan result)

	// Run a infinite loop executing using our runner.
	go func() {
		for {
			time.Sleep(1 * time.Millisecond)

			// Execute concurrently.
			go func() {
				// Execute our call to the service.
				var msg string
				err := runner.Run(context.TODO(), func(ctx context.Context) error {
					now := time.Now()

					// If minute is mod 3 return error directly
					if now.Minute()%3 == 0 {
						return fmt.Errorf("huge system error")
					}

					var err error
					switch time.Now().Nanosecond() % 10 {
					case 0:
						msg = "ok"
					case 2, 9:
						time.Sleep(750 * time.Millisecond)
						err = fmt.Errorf("a error")
					case 7:
						time.Sleep(5 * time.Second)
						msg = "ok"
					default:
						time.Sleep(20 * time.Millisecond)
						if rand.Intn(1000)%2 == 0 {
							msg = "ok"
						} else {
							err = fmt.Errorf("another error")
						}
					}

					return err
				})

				// Send the result to our receiver outside this infinite loop.
				results <- result{
					err: err,
					msg: msg,
				}
			}()

		}
	}()

	// Process the received executions.
	for res := range results {
		if res.err != nil {
			fmt.Printf("[!] fallback because err received: %s\n", res.err)
		} else {
			fmt.Printf("[*] all ok: %s\n", res.msg)
		}
	}
}
