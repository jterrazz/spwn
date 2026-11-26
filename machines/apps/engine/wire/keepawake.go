package wire

import (
	"context"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	awakecalls "github.com/jterrazz/studio-station/apps/engine/awake/calls"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagekeepawake "github.com/jterrazz/studio-station/apps/engine/pages/keepawake"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	settingscalls "github.com/jterrazz/studio-station/apps/engine/settings/calls"
)

func (m machine) keepAwakePage(v awake.View) any {
	r := awake.Report{Now: m.sys.Clock.Now(), Zone: time.Local, Engine: awake.EngineRunning, Reading: v.Reading, Session: v.Session, LastEnd: v.LastEnd, Guards: v.Guards}
	return page.Document{Schema: page.Schema, Page: pagekeepawake.Page(r, m.where)}
}

func awakeDown(err error) error { return downAs(err, awake.ErrEngineDown) }

func (m machine) awakeWaker() *awakecalls.Waker {
	c := m.engineAPI()
	return awakecalls.NewWaker(m.sys.Clock, awakecalls.WakerPorts{
		View: func(ctx context.Context) (awake.View, error) {
			var v awake.View
			err := c.do(ctx, engine.CallAwakeView, nil, &v)
			return v, awakeDown(err)
		},
		Awake: func(ctx context.Context, r awake.Request) (awake.Awoken, error) {
			var a awake.Awoken
			in := engine.AwakeIn{Lid: r.Lid, Battery: r.Battery, For: int(r.For / time.Second), By: r.By}
			if !r.Until.IsZero() {
				in.Until = &r.Until
			}
			err := c.do(ctx, engine.CallKeepAwake, in, &a)
			return a, awakeDown(err)
		},
		Sleep: func(ctx context.Context) (awake.Slept, error) {
			var s engine.AwakeOff
			err := c.do(ctx, engine.CallAwakeOff, nil, &s)
			return awake.Slept{Ended: s.Ended, After: time.Duration(s.After) * time.Second}, awakeDown(err)
		},
		Installed: m.engineInstalled,
	}, awakecalls.WakerOptions{Power: m.powerPort, ThisMachine: m.kind, ThisMacName: m.where.Machine, Zone: time.Local, MacNeverSleeps: m.sleep.Fresh})
}

// value: this Mac's own sleep timer reads off, within the settings' read
// budget; false on a Linux machine or a Mac that cannot say.
func (m machine) readMacNeverSleeps(ctx context.Context) bool {
	if m.kind != catalog.Mac {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, settingscalls.ReadBudget)
	defer cancel()
	got, err := m.facts().Read(ctx, catalog.FactSystemSleep)
	return err == nil && settings.NeverSleeps(got)
}
