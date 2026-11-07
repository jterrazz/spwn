package process

import (
	"context"
	"net/netip"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/engine"
	model "github.com/jterrazz/studio-station/src/machines"
)

type CheckRunner interface {
	// invariant: what the engine needs to run a server's check from its
	// Mac: nil once it answered within its budget, with what it answered.
	Run(ctx context.Context, t model.CheckTarget) (string, error)
}

// invariant: what the engine tells goes to the Notifier and to every open
// watch alike, its parts joined as the presentation joins them.
func (d *Engine) tell(ctx context.Context, vm string, t engine.Told) {
	text := strings.Join(t.Parts, ", ")
	if d.o.Join != nil {
		text = d.o.Join(t.Parts...)
	}
	d.notify(ctx, engine.Notification{Kind: t.Kind, Subject: vm, Text: text})
	d.hub.tell(engine.Notice{Kind: t.Kind, Text: text})
}

// invariant: a server whose run ended unexpectedly boots again after its
// wait (SPEC §7.4); the owner is told of each restart, and once of a loop.
func (d *Engine) restartLater(ctx context.Context, v *vmState, up time.Duration, now time.Time) {
	d.mu.Lock()
	was := v.restarts.Looping
	v.restarts = v.restarts.After(up, now)
	r := v.restarts
	v.rerunAt = now.Add(r.Wait)
	v.state, v.detail = engine.StateStopped, restartingDetail
	d.mu.Unlock()
	switch {
	case r.Looping && !was:
		d.tell(ctx, v.rec.Name, engine.Looping(v.rec.Name, r))
	case !r.Looping:
		d.tell(ctx, v.rec.Name, engine.Restarting(v.rec.Name, r.Count))
	}
}

// invariant: a running server's health check runs after its boot once it
// has an address, then every minute, never two at once; a loop ends once a
// run stays up 10 min. A failing check never restarts it.
func (d *Engine) keep(ctx context.Context, v *vmState) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if v.rec.Lifetime != model.AlwaysOn || v.run == nil || v.stopping || v.state != engine.StateRunning {
		return
	}
	now := d.h.Now()
	v.restarts = v.restarts.Steady(now.Sub(v.bootAt), now)
	check, ok := model.ParseCheck(v.rec.Check)
	addr, err := netip.ParseAddr(v.ip)
	if d.o.Checks == nil || !ok || err != nil || v.checking || !v.check.Due(now) {
		return
	}
	v.checking = true
	target := check.Target(v.rec.Name, addr)
	if d.life != nil {
		ctx = d.life
	}
	go func() {
		answer, err := d.o.Checks.Run(ctx, target)
		passed := err == nil
		d.mu.Lock()
		var change model.CheckChange
		v.check, change = model.Health(v.check, passed, now)
		if passed {
			v.check.Answer = answer
		}
		v.checking = false
		d.mu.Unlock()
		switch change {
		case model.CheckFails:
			d.logf(v.rec.Name, "its check fails: "+v.rec.Check)
		case model.CheckBack:
			d.tell(ctx, v.rec.Name, engine.Back(v.rec.Name))
		case model.CheckSame:
		}
	}()
}

func (d *Engine) Events(ctx context.Context, send func(any) error) error {
	// invariant: the events watch streams each event of the closed list the
	// engine tells, once, as it is told, until the watcher leaves (the menu
	// bar's notifications); any other word it says is not one.
	sub := d.hub.add()
	defer d.hub.remove(sub)
	if send(engine.Opened{Stream: engine.EventsStream}) != nil {
		return nil
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case n := <-sub.notice:
			if !engine.Notifies(n.Kind) {
				continue
			}
			if err := send(engine.Event{Event: "notice", Notice: &n}); err != nil {
				return nil
			}
		}
	}
}
