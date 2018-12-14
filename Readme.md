# Goresilience

Goresilience is a Go toolkit to increase the resilence of applications. Inspired by hystrix and similar libraries at it's core but at the same time very different:

## Features

- Increase resilence of the programs.
- Easy to extend, test and with clean desing.
- Go idiomatic.
- Use the decorator pattern (middleware), like Go's http.Handler does.
- Ability to create custom resilence flows, simple, advanced, specific... by combining different runners in chains.
- Safety defaults.
- Not couple to any framework/library.
- Prometheus/Openmetrics metrics as first class citizen.

## Motivation

You are wondering, why another circuit breaker library...?

Well, this is not a circuit breaker library. Is true that Go has some good circuit breaker libraries (like [sony/gobreaker], [afex/hystrix-go] or [rubyist/circuitbreaker]). But there is a lack a resilience toolkit that is easy to extend, customize and stablishes a design that can be extended, that's why goresilience born.

The aim of goresilience is to use the library with the resilience runners that can be combined or used independently depending on the execution logic nature (complex, simple, performance required, very reliable...).

Also one of the key parts of goreslience is the extension to create new runners yourself and use it in combination with the bulkhead, the circuitbreaker or any of the runners of this library or from others.

## Getting started

The usage of the library is simple. Everything is based on `Runner` interface.

The runners can be used in two ways, in standalone mode (one runner):

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/slok/goresilience/timeout"
)

func main() {
    // Create our command.
    cmd := timeout.New(timeout.Config{
        Timeout: 100 * time.Millisecond,
    })

    for i := 0; i < 200; i++ {
        // Execute.
        result := ""
        err := cmd.Run(context.TODO(), func(_ context.Context) error {
            if time.Now().Nanosecond()%2 == 0 {
                time.Sleep(5 * time.Second)
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

```

or combining in a chain of multiple runners by combining runner middlewares. In this example the execution will be retried timeout and concurrency controlled using a runner chain:

```go
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
```

As you see, you could create any combination of resilient execution flows by combining the different runners of the toolkit.

## Resilience `Runner`s

### Static

Static runners are the ones that based on a static configuration and don't change based on the environment (unlike the adaptive ones).

#### Timeout

This runner is based on timeout pattern, it will execute the `goresilience.Func` but if the execution duration is greater than a T duration timeout it will return a timeout error.

Check [example][timeout-example].

#### Retry

This runner is based on retry pattern, it will retry the execution of `goresilience.Func` in case it failed N times.

It will use a exponential backoff with some jitter (for more information check [this][amazon-retry])

Check [example][retry-example].

#### Bulkhead

This runner is based on [bulkhead pattern][bulkhead-pattern], it will control the concurrecy of `goresilience.Func` executions using the same runner.

It also can timeout if a `goresilience.Func` has been waiting too much to be executed on a queue of execution.

Check [example][bulkhead-example].

#### Circuit breaker

This runner is based on [circuitbreaker pattern][circuit-breaker-url], it will be storing the results of the executed `goresilience.Func` in N buckets of T time to change the state of the circuit based on those measured metrics.

Check [example][circuitbreaker-example].

#### Chaos

This runner is based on [failure injection][chaos-engineering] of errors and latency. It will inject those failures on the required executions (based on percent or all).

Check [example][chaos-example].

### Adaptive

TODO

### Other

#### Metrics

All the runners can be measured using a `metrics.Recorder`, but instead of passing to every runner, the runners will try to get this recorder from the context. So you can wrap any runner using `metrics.NewMiddleware` and it will activate the metrics support on the wrapped runners. This should be the first runner of the chain.

At this moment only [Prometheus][prometheus-url] is supported.

In this [example][hystrix-example] the runners are measured.

Measuring has always a performance hit (not too high), on most cases is not a problem, but there is a benchmark to see what are the numbers:

```text
BenchmarkMeasuredRunner/Without_measurement_(Dummy).-4            300000              6580 ns/op             677 B/op         12 allocs/op
BenchmarkMeasuredRunner/With_prometheus_measurement.-4            200000             12901 ns/op             752 B/op         15 allocs/op
```

#### Hystrix-like

Using the different runners a hystrix like library flow can be obtained. You can see a simple example of how it can be done on this [example][hystrix-example]

## Architecture

At its core, goresilience is based on a very simple idea, the `Runner` interface, `Runner` interface is the unit of execution, its accepts a `context.Context`, a `goresilience.Func` and returns an `error`.

The idea of the Runner is the same as the go's `http.Handler`, having a interface you could create chains of runners, also known as middlewares (Also called decorator pattern).

The library comes with decorators called `Middleware` that return a function that wraps a runner with another runner and gives us the ability to create a resilient execution flow having the ability to wrap any runner to customize with the pieces that we want including custom ones not in this library.

This way we could create execution flow like this example:

```text
Circuit breaker
└── Timeout
    └── Retry
```

## Extend using your own runners

To create your own runner, You need to have 2 things in mind.

- Implement the `goresilience.Runner` interface.
- Give constructors to get a `goresilience.Middleware`, this way your `Runner` could be chained with other `Runner`s.

In this example (full example [here][extend-example]) we create a new resilience runner to make chaos engeniering that will fail at a constant rate set on the `Config.FailEveryTimes` setting.

Following the library convention with `NewFailer` we get the standalone Runner (the one that is not chainable). And with `NewFailerMiddleware` We get a `Middleware` that can be used with `goresilience.RunnerChain` to chain with other Runners.

Note: We can use `nil` on `New` because `NewMiddleware` uses `goresilience.SanitizeRunner` that will return a valid Runner as the last part of the chain in case of being `nil` (for more information about this check `goresilience.command`).

```golang
// Config is the configuration of constFailer
type Config struct {
    // FailEveryTimes will make the runner return an error every N executed times.
    FailEveryTimes int
}

// New is like NewFailerMiddleware but will not wrap any other runner, is standalone.
func New(cfg Config) goresilience.Runner {
    return NewFailerMiddleware(cfg)(nil)
}

// NewMiddleware returns a new middleware that will wrap runners and will fail
// evey N times of executions.
func NewMiddleware(cfg Config) goresilience.Middleware {
    return func(next goresilience.Runner) goresilience.Runner {
        calledTimes := 0
        // Use the RunnerFunc helper so we don't need to create a new type.
        return goresilience.RunnerFunc(func(ctx context.Context, f goresilience.Func) error {
            // We should lock the counter writes, not made because this is an example.
            calledTimes++

            if calledTimes == cfg.FailEveryTimes {
                calledTimes = 0
                return fmt.Errorf("failed due to %d call", calledTimes)
            }

            // Run using the the chain.
            next = runnerutils.Sanitize(next)
            return next.Run(ctx, f)
        })
    }
}
```

[sony/gobreaker]: https://github.com/sony/gobreaker
[afex/hystrix-go]: https://github.com/afex/hystrix-go
[rubyist/circuitbreaker]: https://github.com/rubyist/circuitbreaker
[circuit-breaker-url]: https://martinfowler.com/bliki/CircuitBreaker.html
[retry-example]: examples/retry
[timeout-example]: examples/timeout
[bulkhead-example]: examples/bulkhead
[circuitbreaker-example]: examples/circuitbreaker
[chaos-example]: examples/chaos
[hystrix-example]: examples/hystrix
[extend-example]: examples/extend
[amazon-retry]: https://aws.amazon.com/es/blogs/architecture/exponential-backoff-and-jitter/
[bulkhead-pattern]: https://docs.microsoft.com/en-us/azure/architecture/patterns/bulkhead
[chaos-engineering]: https://en.wikipedia.org/wiki/Chaos_engineering
[prometheus-url]: http://prometheus.io
