package circuit

import "context"

var (
	contextKeyCommand = contextKey("command")
)

// CommandNameFromContext returns the command name from the context, and
// a ok boolean is not present.
func CommandNameFromContext(ctx context.Context) (cmd string, ok bool) {
	cmd, ok = ctx.Value(contextKeyCommand).(string)
	return cmd, ok
}

// NewNamed will set the name to the command of execution.
func NewNamed(name string, cb Breaker) Breaker {
	return BreakerFunc(func(ctx context.Context) (fallback bool, err error) {
		ctx = context.WithValue(ctx, contextKeyCommand, name)
		return cb.Run(ctx)
	})
}

// Command is the unit of executon of a Circuit Breaker.
type Command func(ctx context.Context) error

// Run will execute the command.
func (c Command) Run(ctx context.Context) (fallback bool, err error) {
	// Only execute if we reached to the execution and the context has not been cancelled.
	select {
	case <-ctx.Done():
		return true, nil
	default:
		return false, c(ctx)
	}
}
