package process

import (
	"context"
	"errors"
	"time"

	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/catalog"

	awakecalls "github.com/jterrazz/studio-station/src/awake/calls"
	"github.com/jterrazz/studio-station/src/engine"
)

func (d *Engine) newAwake() *awakecalls.Keeper {
	return awakecalls.NewKeeper(d.o.Power, d.h, awakecalls.KeeperOptions{
		Log:     func(m string) { d.logf("awake", m) },
		Notice:  func(e awake.End) { d.told(engine.NoticeAwakeEnded, engine.AwakeStopped(e.Detail)) },
		Told:    func(text string) { d.told(engine.NoticeNeverSleeps, text) },
		Change:  d.hub.changed,
		Working: d.working,
	})
}

// invariant: every end of Awake and every word of Never sleeps is told once,
// to the open surfaces and to the owner's notifications.
func (d *Engine) told(kind, text string) {
	d.hub.tell(engine.Notice{Kind: kind, Text: text})
	d.notify(context.Background(), engine.Notification{Kind: kind, Subject: "awake", Text: text})
}

// invariant: at its start the engine holds Never sleeps before any VM
// starts, as held since that start.
func (d *Engine) neverSleeps(since time.Time) {
	if d.o.NeverSleeps != nil {
		d.awakeJob.Policy(d.o.NeverSleeps(), since)
	}
}

func (d *Engine) AwakeKept() awake.Kept {
	// value: what this engine keeps of Awake, for the stepped engine that
	// follows it.
	return d.awakeJob.Kept()
}

func (d *Engine) RememberAwake(k awake.Kept) { d.awake = &k }

func (d *Engine) Suspend() {
	d.awakeJob.Suspend()
	d.closeAll()
}

func (d *Engine) watchAwake(ctx context.Context) {
	if d.o.Power == nil {
		return
	}
	for ev := range d.o.Power.Watch(ctx) {
		d.awakeJob.Observe(ctx, ev)
	}
}

func (d *Engine) keepAwake(r engine.AwakeIn) (awake.Awoken, error) {
	req := awake.Request{Lid: r.Lid, Battery: r.Battery, For: time.Duration(r.For) * time.Second, By: r.By, Run: r.Run}
	if r.Until != nil {
		req.Until = *r.Until
	}
	session, refreshed, err := d.awakeJob.Awake(req)
	s := awake.Awoken{Session: session, Refreshed: refreshed}
	var rng *awake.RangeError
	var guard *awake.GuardError
	var none *awake.UnsupportedError
	var run *awake.RunError
	switch {
	case errors.As(err, &run):
		return s, fault("awake_run", run.Error(), "station awake 4h")
	case errors.As(err, &rng):
		return s, fault("awake_range", rng.Error(), "station awake 4h")
	case errors.As(err, &guard):
		hint := "station awake"
		if guard.Guard == awake.ReasonLid {
			hint = "station awake 4h --lid --battery"
		}
		return s, fault("awake_guard", guard.Error(), hint)
	case errors.As(err, &none):
		return s, fault("awake_unsupported", none.Error(), catalog.AwakeOnItsMac)
	}
	return s, err
}

func (d *Engine) endAwake() engine.AwakeOff {
	s := d.awakeJob.Sleep()
	return engine.AwakeOff{Ended: s.Ended, After: int(s.After.Seconds())}
}
