package retry

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/slok/goresilience"
	"github.com/slok/goresilience/metrics"
)

// Config is the configuration used for the retry Runner.
type Config struct {
	// WaitBase is the base unit duration to wait on the retries.
	WaitBase time.Duration
	// Backoff enables exponential backoff on the retry (also disables jitter).
	DisableBackoff bool
	// Times is the number of times that will be retried in case of error
	// before returning the error itself.
	Times int
}

const (
	defaultWaitBase     = 20 * time.Millisecond
	defaultTimesToRetry = 3
)

func (c *Config) defaults() {
	if c.WaitBase <= 0 {
		c.WaitBase = defaultWaitBase
	}

	// TODO: Allow -1 for forever retries?
	if c.Times <= 0 {
		c.Times = defaultTimesToRetry
	}
}

// New returns a new retry ready executor, the execution will be retried the number
// of times specified on the config (+1, the original execution that is not a retry).
func New(cfg Config) goresilience.Runner {
	return NewMiddleware(cfg)(nil)
}

// NewMiddleware returns a new retry middleware, the execution will be retried the number
// of times specified on the config (+1, the original execution that is not a retry).
func NewMiddleware(cfg Config) goresilience.Middleware {
	cfg.defaults()

	return func(next goresilience.Runner) goresilience.Runner {
		next = goresilience.SanitizeRunner(next)
		random := rand.New(rand.NewSource(time.Now().UnixNano()))

		// Use the algorithms for jitter and backoff.
		// https://aws.amazon.com/es/blogs/architecture/exponential-backoff-and-jitter/
		return goresilience.RunnerFunc(func(ctx context.Context, f goresilience.Func) error {
			var err error
			metricsRecorder, _ := metrics.RecorderFromContext(ctx)

			// Start the attempts (it's 1 + the number of retries.)
			for i := 0; i <= cfg.Times; i++ {
				// Only measure the retries.
				if i != 0 {
					metricsRecorder.IncRetry()
				}

				err = next.Run(ctx, f)
				if err == nil {
					return nil
				}

				// We need to sleep before making a retry.
				waitDuration := cfg.WaitBase

				// Apply Backoff.
				// The backoff is calculated exponentially based on a base time
				// and the attempt of the retry.
				if !cfg.DisableBackoff {
					exp := math.Exp2(float64(i))
					waitDuration = time.Duration(float64(cfg.WaitBase) * exp)
					// Apply "full jitter".
					waitDuration = time.Duration(float64(waitDuration) * random.Float64())

					waitDuration = waitDuration.Round(time.Millisecond)
				}

				time.Sleep(waitDuration)
			}

			return err
		})
	}
}
