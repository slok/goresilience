package limit

import (
	"time"
)

// Result is the result kind to be measured by the Limiter algorithm. These
// results are the ones the algorithm will be taken into account to
// calculate the dynamic limits.
type Result string

const (
	// ResultSuccess will be treated like a success by the algorithm.
	ResultSuccess Result = "success"
	// ResultFailure will be treated like a Failure by the algorithm.
	ResultFailure Result = "failure"
	// ResultIgnore will be ignored by the algorithm.
	ResultIgnore Result = "ignore"
)

// Limiter knows what should be the concurrency limit based on the measured results.
// These are based on TCP congestion control algorithms.
type Limiter interface {
	// MeasureSample will measure the sample of an execution. This data will be used
	// by the algorithm to know what should be the limit.
	// It also returns the current limit after measuring the samples.
	MeasureSample(startTime time.Time, inflight int, result Result) int

	// Gets the current limit.
	GetLimit() int
}
