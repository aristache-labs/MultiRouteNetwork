package engine

import (
	"context"
	"time"
)

// FlowHint is what the desktop knows about the process that opened a flow.
// OnBytes may sleep to apply a per-app limit. It must not call back into the scheduler.
type FlowHint struct {
	OnOpen func(adapter string) func()

	Name, Path string
	Known      bool
	Blocked    bool
	DNS        string
	OnBytes    func(adapter string, down bool, n int, udp bool)
	Stop       func() bool
}

type hintKey struct{}

// WithHint attaches a flow hint to a dial.
func WithHint(ctx context.Context, h FlowHint) context.Context {
	return context.WithValue(ctx, hintKey{}, h)
}

// HintFrom returns the hint stored by WithHint.
func HintFrom(ctx context.Context) FlowHint {
	if ctx == nil {
		return FlowHint{}
	}
	h, _ := ctx.Value(hintKey{}).(FlowHint)
	return h
}

// AdapterView is one candidate adapter, passed to a Policy while the scheduler lock is held.
type AdapterView struct {
	Name    string
	Up      bool
	Latency time.Duration
}

// Decision tells AcquireForHint how to narrow the candidate pool.
// Block drops the flow. Pin uses one adapter. Only balances inside that set.
// An empty decision keeps the original balance.
type Decision struct {
	Block bool
	Pin   string
	Only  []string
}

// Policy chooses an adapter for one new flow. It runs under the scheduler lock.
type Policy func(hint FlowHint, adapters []AdapterView) Decision
