package concurrencylimit

// concurrencylimit it's inspired by the implementation of Netflix/concurrency-limits
// it tries to create a concurrency controlled execution flow using adaptive algorithms
// from TCP congestion control algorithms.
//
// The implementation of concurrencylimit runner is based in 4 componentes:
//
// - The Limiter algorithm: This is the algorithm used to calculate
// 	 the limit of concunrrency. There are multiple to select based on the type
// 	 of execution flow used on the Runner (client side, server side,
//   lot's of dependencies...).
//
// - The Executor: The executor is the one that knows how to execute Funcs, it
// 	 controls the concurrency, increases and decreases the number of executor
//   workers. There are different kinds like blockers, partitioned...
//	 One important thing is that the executors sometimes (depends on the implementation)
//   will return a execution rejected exception (errors.ErrLimiterExecutionRejected),
//   this doesn't count for the algorithm and could be checked by the user to do another
//	 thing, fallback, return error or whatever.
//
// - The Runner: This is the one that implements goresilience.Runner, it's the
//	 one that will be used directly by a goreslience library user. It glues
//	 together the algorithm and the executor, knows how to handle the  user Func
//   to be executed by the executor, knows how to measure the result to give it
//   to the algorithm to create a new limit dynamically, sets those limits on the
//	 executor...
//
// - The result policy: The concurrencylimit Runner has a configuration field that is the error policy
//   this policy is the one responsible to tell if the received error by the function
//   is a real error for the limit algorithm or not. By default every external error to
//   concurrencylimit reject error is treated as a failure and will count on the algorithm
//   as an failure.
//   Another policy could be only the errors that are not of X type and the field Y
//   from the context has Z attribute to true.
