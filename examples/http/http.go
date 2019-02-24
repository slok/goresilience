// This example will show how a concurrency limit runner can be used to control
// the request handling limits and protect the server by using a middleware.
// Use some tool like Vegeta to test the load of the server and see how much
// is load shedded by the goresilience runner.
// example:  `echo "GET http://127.0.0.1:8080" | vegeta attack -rate=XXX/s -duration=YYYm | vegeta report`

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/slok/goresilience"
	"github.com/slok/goresilience/concurrencylimit"
	"github.com/slok/goresilience/concurrencylimit/execute"
	"github.com/slok/goresilience/concurrencylimit/limit"
)

const (
	addr      = ":8080"
	sleepTime = 640 * time.Millisecond
)

func main() {
	h := concurrencylimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// "fake" work by sleeping.
		time.Sleep(sleepTime)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "request handled at %s", time.Now())
	}))

	log.Printf("serving at %s", addr)
	http.ListenAndServe(addr, h)
}

func concurrencylimitMiddleware(next http.Handler) http.Handler {
	// Create our resilience pattern using a goresilience concurrency limiter.
	runner := goresilience.RunnerChain(
		concurrencylimit.NewMiddleware(concurrencylimit.Config{
			Executor: execute.NewLIFO(execute.LIFOConfig{}),
			Limiter:  limit.NewAIMD(limit.AIMDConfig{}),
			// We don't want to adapt based on loss, instead use latency.
			ExecutionResultPolicy: concurrencylimit.NoFailurePolicy,
		}),
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := runner.Run(r.Context(), func(_ context.Context) error {
			next.ServeHTTP(w, r)
			return nil
		})

		// Notify about the load shedding by setting a 429.
		if err != nil {
			w.WriteHeader(http.StatusTooManyRequests)
		}
	})
}
