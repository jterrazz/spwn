package wire

import (
	"context"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	awakecalls "github.com/jterrazz/studio-station/apps/engine/awake/calls"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagekeepawake "github.com/jterrazz/studio-station/apps/engine/pages/keepawake"
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
			in := engine.AwakeIn{Lid: r.Lid, Battery: r.Battery, For: int(r.For / time.Second), By: r.By, Run: r.Run}
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
		Installed: func() bool {
			_, err := m.sys.FS.Stat(catalog.EnginePlist(m.sys.Env.Home()))
			return err == nil
		},
	}, awakecalls.WakerOptions{Power: m.powerPort, ThisMachine: m.kind, ThisMacName: m.where.Machine, Zone: time.Local})
}

type statusAwake struct{ s *awakecalls.Waker }

func (p statusAwake) Session(ctx context.Context) (overview.AwakeState, bool) {
	if p.s.Check() != nil {
		return overview.AwakeState{}, false
	}
	var last awake.Report
	for r := range p.s.Read(ctx) {
		last = r
	}
	if last.Engine != awake.EngineRunning {
		return overview.AwakeState{}, false
	}
	if last.Session == nil {
		return overview.AwakeState{}, true
	}
	return overview.AwakeState{Held: true, Lid: last.Session.Lid, Until: last.Session.Until}, true
}
