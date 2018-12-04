# Goresilience

Goresilience is a toolkit to increase the resilence of applications for go. Inspired by hystrix at it's core but at the same time very different:

The aim of this library are:

- Increase resilence of the programs.
- Easy to extend, test and with clean desing.
- Go idiomatic.
- Empower the decorator pattern (middleware, wrapper), like Go's http.Handler does.
- Ability to create custom resilence flows, simpled, advanced, specific...
- Safety defaults and ready with the most common execution flows.
- Allow extending resilence of a exeucution flow from other repositories (like http.Handler middlewares).
- Not couple to any framework/library.

## Motivation

Go has some good circuit breaker libraries (like [gobreaker], [hystrix-go] or [circuitbreaker])

But there is a lack a resilience toolkit that is easy to extend and customize, that's why goresilience born.

## Architecture

At its core, goresilience is based on a very simple idea, the `Runner` interface, `Runner` interface is the unit of execution, its accepts a `context.Context` and returns an `error`. The library comes with decorators that implement this interface and gives us the ability to create a resilient execution flow, customize with the pieces that we want and create our custom ones.

Example of a full resilient flow using some of the toolkit parts:

```text
Circuit breaker
└── Timeout
    └── Retry
```

## Toolkit

- [timeout]
  - static: Static timeout timeouts using a static duration.

[gobreaker]: https://github.com/sony/gobreaker
[hystrix-go]: https://github.com/afex/hystrix-go
[circuitbreaker]: https://github.com/rubyist/circuitbreaker
[timeout]: timeout/
