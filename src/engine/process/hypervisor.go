package process

import (
	"context"
	"net/netip"
	"time"

	model "github.com/jterrazz/studio-station/src/machines"
)

type Hypervisor interface {
	// invariant: what the engine needs of a VM's hypervisor: a run it boots or
	// adopts, and how that run ended; never how the hypervisor runs it.
	Boot(ctx context.Context, vm string, n model.Network) (Run, error)
	Adopt(ctx context.Context, vm string) (Run, bool)
	Shutdown(ctx context.Context, vm string, grace time.Duration) error
	Address(ctx context.Context, vm string, wait time.Duration) (netip.Addr, error)
	Console(ctx context.Context, vm string, n int) ([]string, error)
}

type Run interface {
	VM() string
	PID() int
	Holder() int
	Ended() <-chan model.RunEnd
}

var unread = model.RunEnd{Reason: model.Unknown, ExitCode: -1}

// invariant: over keeps the end it read, so whoever asks first, the
// supervisor still reads it; the caller holds the daemon's lock.
func (v *vmState) over() bool {
	if v.run == nil || v.end != nil {
		return true
	}
	select {
	case end, ok := <-v.run.Ended():
		if !ok {
			end = unread
		}
		v.end = &end
		return true
	default:
		return false
	}
}

func (v *vmState) runPID() int {
	if v.run == nil {
		return 0
	}
	return v.run.PID()
}

func (v *vmState) holder() int {
	if v.run == nil {
		return 0
	}
	return v.run.Holder()
}

func (d *Engine) runs(v *vmState) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return v.run != nil && !v.over()
}
