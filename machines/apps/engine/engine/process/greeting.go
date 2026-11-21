package process

import (
	"context"
	"encoding/json"
)

// invariant: a VM's guest greets from the moment its session is dialed until
// it has said it is ready and sent its first scan, or until that session
// ends: what the guest says at the start is in the engine once none greets.
type greetings struct {
	n    int
	none chan struct{}
}

func (d *Engine) greetLocked(v *vmState) {
	if v.greeting {
		return
	}
	v.greeting, v.greeted = true, make(chan struct{})
	if d.greets.n == 0 {
		d.greets.none = make(chan struct{})
	}
	d.greets.n++
}

func (d *Engine) greetedLocked(v *vmState) {
	if !v.greeting {
		return
	}
	v.greeting = false
	close(v.greeted)
	d.greets.n--
	if d.greets.n == 0 {
		close(d.greets.none)
	}
}

// invariant: only the session the VM holds now ends its greeting: a line or
// an end of a session it no longer holds leaves the new one greeting.
func (d *Engine) greetedBy(v *vmState, gen int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if v.gen == gen {
		d.greetedLocked(v)
	}
}

func (d *Engine) fromSession(ctx context.Context, v *vmState, gen int, line []byte) {
	d.fromGuest(ctx, v, line)
	d.mu.Lock()
	greeting := v.greeting && v.gen == gen
	d.mu.Unlock()
	var m guestMessage
	if greeting && json.Unmarshal(line, &m) == nil && m.Type == "snapshot" {
		d.greetedBy(v, gen)
	}
}

// value: closed once v's guest ends the greeting it is in; nil while it
// greets none.
func (d *Engine) greetingOf(v *vmState) <-chan struct{} {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !v.greeting {
		return nil
	}
	return v.greeted
}

func (d *Engine) Greeted() <-chan struct{} {
	// value: closed once no VM's guest greets the engine.
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.greets.n == 0 {
		none := make(chan struct{})
		close(none)
		return none
	}
	return d.greets.none
}
