# Goresilience

Goresilience is a Go toolkit to increase the resilence of applications. Inspired by hystrix at it's core but at the same time very different:

## Features

- Increase resilence of the programs.
- Easy to extend, test and with clean desing.
- Go idiomatic.
- Empower the decorator pattern (middleware, wrapper), like Go's http.Handler does.
- Ability to create custom resilence flows, simpled, advanced, specific... by combining different pieces
- Safety defaults and ready with the most common execution flows.
- Not couple to any framework/library.
- Prometheus/openmetrics metrics as first class citizen.

## Motivation

You are wondering, why another circuit breaker library...?

Well, this is not a circuit breaker library. Is true that Go has some good circuit breaker libraries (like [sony/gobreaker], [afex/hystrix-go] or [rubyist/circuitbreaker]). But there is a lack a resilience toolkit that is easy to extend and customize, that's why goresilience born.

The aim of goresilience is to use the library with the resilience runners that can be combined or used independently depending on the execution logic nature (complex, simple, performance required, very reliable...).

Also one of the key parts of goreslience is the extension to create new runners yourself and use it in combination with the bulkhead, the circuitbreaker or any of the runners.

## Toolkit

- [bulkhead]
  - static: Static bulkhead controls the execution concurrency by limiting to a specific number of workers.
- [circuitbreaker]: The circuit breaker will fail fast if the execution fails too much, more information about how a circuit breaker works [here][circuit-breaker-url].
- [retry]: will retry the configured number of times using exponential backoff.
- [timeout]
  - static: Static timeout timeouts using a static duration.

## Architecture

At its core, goresilience is based on a very simple idea, the `Runner` interface, `Runner` interface is the unit of execution, its accepts a `context.Context`, a `goresilience.Func` and returns an `error`. The library comes with decorators that implement this interface and gives us the ability to create a resilient execution flow, customize with the pieces that we want and create our custom ones.

Example of a full resilient flow using some of the toolkit parts:

```text
Circuit breaker
└── Timeout
    └── Retry
```

## How to use it

The usage of the library is simple. You have each of the toolkit part factories and you can use it standalone like this example where the execution will be retried if it fails:

```go
import (
    "github.com/slok/goresilience/retry"
)

// Create our execution chain.
cmd = retry.New(retry.Config{}, nil)

// Execute.
calledCounter := 0
err := exec.Run(context.TODO(), func(_ context.Context) error {
    calledCounter++
    if calledCounter % 2 == 0 {
        return errors.New("you didn't expect this error")
    }
    return nil
})
```

or combining in a chain, like this example were the execution will be retried, timeout and concurrency controlled:

```go
import (
    "github.com/slok/goresilience/retry"
    "github.com/slok/goresilience/bulkhead"
    "github.com/slok/goresilience/timeout"
)

// Create our execution chain.
cmd =   bulkhead.NewStatic(bulkhead.StaticConfig{},
            retry.New(retry.Config{},
                timeout.NewStatic(timeout.StaticConfig{}, nil)))

// Execute.
calledCounter := 0

err := exec.Run(context.TODO(), func(_ context.Context) error {
    calledCounter++
    if calledCounter % 2 == 0 {
        return errors.New("you didn't expect this error")
    }
    return nil
})
```

As you see, it's easy to create any combination of resilient execution flows by combining the different runners of the toolkit.

## Create your own resilient runner

You can extend the toolkit by implementing the `goresilience.Runner` interface.

In this exaxmple we create a new resilience runner to make chaos engeniering that will fail at a constant rate set on the `Config.FailEveryTimes` setting:

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
[bulkhead]: bulkhead/
[circuitbreaker]: circuitbreaker/
[retry]: retry/
[timeout]: timeout/
[circuit-breaker-url]: https://martinfowler.com/bliki/CircuitBreaker.html
