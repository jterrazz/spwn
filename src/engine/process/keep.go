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
	// Mac: nil once it answered within its budget, with what it answered
	// and how long it took.
	Run(ctx context.Context, t model.CheckTarget) (model.CheckAnswer, error)
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
	v.state, v.detail = engine.StateStopped, "restarting"
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
			v.check.Answer = answer.Status
			v.latency = model.KeepLatency(v.latency, model.LatencyPoint{At: now, Took: answer.Took})
		}
		v.checking = false
		d.mu.Unlock()
		switch change {
		case model.CheckFails:
			d.logf(v.rec.Name, "its check fails: "+v.rec.Check)
			d.wentDown(ctx, v, model.CheckFailed, false, now)
		case model.CheckBack:
			d.tell(ctx, v.rec.Name, engine.Back(v.rec.Name))
		case model.CheckSame:
		}
	}()
}

// invariant: SPEC B5.2: a server goes down on its record once, when its
// run ends unexpectedly or its check fails since it passed; a restart that
// follows walks the four health steps from the first.
func (d *Engine) wentDown(ctx context.Context, v *vmState, cause string, restarts bool, now time.Time) {
	d.mu.Lock()
	if v.rec.Lifetime != model.AlwaysOn {
		d.mu.Unlock()
		return
	}
	was := v.down
	v.down = true
	if restarts {
		v.walk = &model.RestartWalk{Step: 1, At: now}
	}
	d.mu.Unlock()
	if !was {
		d.markUptime(ctx, model.UptimeMark{Server: v.rec.Name, Change: model.WentDown, At: now, Cause: cause})
	}
}

// invariant: SPEC B5.11: a server's restart shows the step it runs and
// when that step began, until all four pass; a server healthy again is back
// on its record, through a restart when it walked one. The first time this
// engine sees it healthy it marks it too, which starts its record and closes
// an incident an earlier engine left open.
func (d *Engine) walk(ctx context.Context, v *vmState) {
	d.mu.Lock()
	if v.rec.Lifetime != model.AlwaysOn {
		d.mu.Unlock()
		return
	}
	now := d.h.Now()
	running := v.state == engine.StateRunning
	_, hasCheck := model.ParseCheck(v.rec.Check)
	step := model.WalkStep(running, running && v.ip != "", running && (v.guest == engine.GuestConnected || v.ssh), !hasCheck || v.check.State == model.CheckPassing)
	if step != 0 {
		if v.walk != nil && v.walk.Step != step {
			v.walk = &model.RestartWalk{Step: step, At: now}
		}
		d.mu.Unlock()
		return
	}
	walked := v.walk != nil
	if !walked && !v.down && v.marked {
		d.mu.Unlock()
		return
	}
	v.walk, v.down, v.marked = nil, false, true
	d.mu.Unlock()
	d.markUptime(ctx, model.UptimeMark{Server: v.rec.Name, Change: model.CameBack, At: now, Restarted: walked})
}

func (d *Engine) markUptime(ctx context.Context, m model.UptimeMark) {
	if err := d.m.MarkUptime(ctx, m); err != nil {
		d.logf(m.Server, "its uptime was not marked: "+err.Error())
	}
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
