# Goresilience

Goresilience is a Go toolkit to increase the resilence of applications. Inspired by hystrix and similar libraries at it's core but at the same time very different:

## Features

- Increase resilence of the programs.
- Easy to extend, test and with clean desing.
- Go idiomatic.
- Use the decorator pattern (middleware, wrapper), like Go's http.Handler does.
- Ability to create custom resilence flows, simple, advanced, specific... by combining different runners.
- Safety defaults and ready with the most common execution flows.
- Not couple to any framework/library.
- Prometheus/Openmetrics metrics as first class citizen.

## Motivation

You are wondering, why another circuit breaker library...?

Well, this is not a circuit breaker library. Is true that Go has some good circuit breaker libraries (like [sony/gobreaker], [afex/hystrix-go] or [rubyist/circuitbreaker]). But there is a lack a resilience toolkit that is easy to extend and customize, that's why goresilience born.

The aim of goresilience is to use the library with the resilience runners that can be combined or used independently depending on the execution logic nature (complex, simple, performance required, very reliable...).

Also one of the key parts of goreslience is the extension to create new runners yourself and use it in combination with the bulkhead, the circuitbreaker or any of the runners.

## Getting started

The usage of the library is simple. You have each of the toolkit `Runner` factories and you can use it standalone like this example where the execution will be retried if it fails:

```go
import (
    //...
    "github.com/slok/goresilience/retry"
)

//...

// Create our execution chain (nil marks the end of the chain).
cmd := retry.New(retry.Config{}, nil)

// Execute.
calledCounter := 0
result := ""
err := cmd.Run(context.TODO(), func(_ context.Context) error {
    calledCounter++
    if calledCounter % 2 == 0 {
        return errors.New("you didn't expect this error")
    }
    result = "all ok"
    return nil
})

if err != nil {
    result = "not ok, but fallback"
}

//...
```

or combining in a chain, like this example were the execution will be retried timeout and concurrency controlled using a runner chain:

```go
import (
    //...

    "github.com/slok/goresilience/retry"
    "github.com/slok/goresilience/bulkhead"
    "github.com/slok/goresilience/timeout"
)

//...

// Create our execution chain (nil marks the end of the chain).
cmd := bulkhead.New(bulkhead.Config{},
    retry.New(retry.Config{},
        timeout.New(timeout.Config{}, nil)))

// Execute.
calledCounter := 0
result := ""
err := cmd.Run(context.TODO(), func(_ context.Context) error {
    calledCounter++
    if calledCounter % 2 == 0 {
        return errors.New("you didn't expect this error")
    }
    result = "all ok"
    return nil
})

if err != nil {
    result = "not ok, but fallback"
}

//...
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

All the runners can be measured using a `metrics.Recorder`, but instead of passing to every runner, the runners will try to get this recorder from the context. So you can wrap any runner using `metrics.NewMeasuredRunner` and it will activate the metrics support on the wrapped runners. This should be the first runner of the chain.

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

At its core, goresilience is based on a very simple idea, the `Runner` interface, `Runner` interface is the unit of execution, its accepts a `context.Context`, a `goresilience.Func` and returns an `error`. The library comes with decorators that implement this interface and gives us the ability to create a resilient execution flow having the ability to wrap any runner to customize with the pieces that we want including custom ones not in this library.

Example of a full resilient flow using some of the toolkit runners:

```text
Circuit breaker
└── Timeout
    └── Retry
```

## Extend using your own runners

You can extend the toolkit by implementing the `goresilience.Runner` interface.

In this example we create a new resilience runner to make chaos engeniering that will fail at a constant rate set on the `Config.FailEveryTimes` setting:

```golang
import (
    "context"
    "fmt"

    "github.com/slok/goresilience"
    runnerutils "github.com/slok/goresilience/internal/util/runner"
)

// Config is the configuration of constFailer
type Config struct {
    // FailEveryTimes will make the runner return an error every N executed times.
    FailEveryTimes int
}

// New returns a new runner that will fail evey N times of executions.
func New(cfg Config, r goresilience.Runner) goresilience.Runner {
    r = runnerutils.Sanitize(r)

    calledTimes := 0
    // Use the RunnerFunc helper so we don't need to create a new type.
    return goresilience.RunnerFunc(func(ctx context.Context, f goresilience.Func) error {
        // We should lock the counter writes, not made because this is an example.
        calledTimes++

        if calledTimes == cfg.FailEveryTimes {
            calledTimes = 0
            return fmt.Errorf("failed due to %d call", calledTimes)
        }
        return nil
    })
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
[amazon-retry]: https://aws.amazon.com/es/blogs/architecture/exponential-backoff-and-jitter/
[bulkhead-pattern]: https://docs.microsoft.com/en-us/azure/architecture/patterns/bulkhead
[chaos-engineering]: https://en.wikipedia.org/wiki/Chaos_engineering
[prometheus-url]: http://prometheus.io
