package process

import (
	"context"
	"net/netip"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/engine"

	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
)

type Guests interface {
	// invariant: what the engine needs of its one connection per VM: a command
	// inside the VM, the session of its guest, a port forwarded to it, and
	// the connection's end.
	Dial(ctx context.Context, vm string, at netip.AddrPort) (Guest, error)
}

type Guest interface {
	Exec(ctx context.Context, line, stdin string) (engine.GuestRan, error)
	// invariant: what the engine writes to the session is the command's
	// input, what it reads its output; the output ends with the command, the
	// connection or ctx.
	Attach(ctx context.Context, line string) (Stream, error)
	Forward(ctx context.Context, ip string, port int) (Stream, error)
	Close() error
	Done() <-chan struct{}
}

func (d *Engine) GuestExec(ctx context.Context, vm, line, stdin string) (engine.GuestRan, error) {
	// invariant: a command inside a VM always rides the engine's own connection
	// to it; with none up, it is refused as absent, never run another way.
	v, err := d.find(vm)
	if err != nil {
		return engine.GuestRan{}, err
	}
	client, state := d.dialed(ctx, v)
	if client == nil && state == engine.StateStopped {
		return engine.GuestRan{}, fault(enginecalls.VMStopped, vm+" is stopped", "station machines start "+vm)
	}
	if client == nil {
		return engine.GuestRan{}, fault("guest_absent", vm+"'s guest is not connected", "station machines start "+vm)
	}
	return client.Exec(ctx, line, stdin)
}

const guestWait = 10 * time.Second

func (d *Engine) dialed(ctx context.Context, v *vmState) (client Guest, state string) {
	// invariant: a call that lands while the guest is being dialed waits for
	// that dial, within guestWait, rather than being refused as absent.
	d.mu.Lock()
	client, state, up := v.client, v.state, v.up
	d.mu.Unlock()
	if client != nil || up == nil {
		return client, state
	}
	wait, cancel := context.WithTimeout(ctx, guestWait)
	defer cancel()
	select {
	case <-up:
	case <-wait.Done():
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return v.client, v.state
}
