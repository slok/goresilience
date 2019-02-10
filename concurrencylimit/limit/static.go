package limit

import (
	"time"
)

// static is an static algorithm that is used for testing purposes, isn't adaptive
// it will have a static limit.
type static struct {
	limit int
}

// NewStatic returns a new Static algorithm that is used ofr testing purposes, isn't adaptive
// it will have a static limit.
func NewStatic(limit int) Limiter {
	return &static{
		limit: limit,
	}
}

// MeasureSample satisfies Algorithm interface.
func (s *static) MeasureSample(_ time.Time, _ int, _ Result) int {
	return s.GetLimit()
}

// GetLimit satsifies Algorithm interface.
func (s *static) GetLimit() int {
	return s.limit
}
