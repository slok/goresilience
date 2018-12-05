package runner

import (
	"github.com/slok/goresilience"
)

// Sanitize returns a safe Runner if the runner is wrong.
func Sanitize(r goresilience.Runner) goresilience.Runner {
	// In case of end of execution chain.
	if r == nil {
		return &goresilience.Command{}
	}
	return r
}
