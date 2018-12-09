package goresilience_test

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/slok/goresilience/retry"
	"github.com/slok/goresilience/timeout"
)

func myFunc(ctx context.Context) error { return nil }

// Will use only one of the utilities, the retry with the default settings
// this will make the `gorunner.Func` to be executed and retried N times if it fails.
func Example_basic() {
	// Create our func `runner`. Use nil as it will not be chained with another `Runner`.
	cmd := retry.New(retry.Config{}, nil)

	// Execute.
	var result string
	err := cmd.Run(context.TODO(), func(ctx context.Context) error {
		resp, err := http.Get("https://bruce.wayne.is.batman.io")
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		result = string(b)
		return nil
	})

	// We could fallback to get a Hystrix like behaviour.
	if err != nil {
		result = "fallback result"
	}

	fmt.Printf("result is: %s\n", result)
}

// Will use more than one `goresilience.Runner` and chain them to create a very
// resilient execution of the `goresilience.Func`.
// In this case we will create a runner that retries and also times out. And we will configure the
// timeout.
func Example_chain() {
	// Create our chain, first the retry and then the timeout with 100ms.
	cmd := retry.New(retry.Config{},
		timeout.NewStatic(timeout.StaticConfig{
			Timeout: 100 * time.Millisecond,
		}, nil))

	var result string
	err := cmd.Run(context.TODO(), func(ctx context.Context) error {
		resp, err := http.Get("https://bruce.wayne.is.batman.io")
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		result = string(b)
		return nil
	})

	// We could fallback to get a Hystrix like behaviour.
	if err != nil {
		result = "fallback result"
	}

	fmt.Printf("result is: %s\n", result)
}

// Is an example to show that when the result is not needed we don't need to
// use and inline function.
func Example_noresult() {
	cmd := retry.New(retry.Config{}, nil)

	// Execute.
	err := cmd.Run(context.TODO(), myFunc)
	if err != nil {
		// Do fallback.
	}
}

// Is an example to show that we could use objects aslo to pass parameter and get our results.
func Example_structresult() {
	type myfuncResult struct {
		name     string
		lastName string
		result   string
	}

	cmd := retry.New(retry.Config{}, nil)

	// Execute.
	res := myfuncResult{
		name:     "Bruce",
		lastName: "Wayne",
	}
	err := cmd.Run(context.TODO(), func(ctx context.Context) error {
		if res.name == "Bruce" && res.lastName == "Wayne" {
			res.result = "Batman"
		}
		return errors.New("identity unknown")
	})

	if err != nil {
		res.result = "Unknown"
	}

	fmt.Printf("%s %s is %s", res.name, res.lastName, res.result)
}
