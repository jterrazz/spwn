package process

import (
	"context"
	"sort"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/engine"
)

type Tick func(ctx context.Context)

func (d *Engine) History() *activity.History {
	// value: the activity this engine keeps of every machine it sees, since
	// it started; nil before Begin.
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.history
}

func (d *Engine) Running() []engine.Running {
	d.mu.Lock()
	defer d.mu.Unlock()
	var out []engine.Running
	for name, v := range d.vms {
		if v.state != engine.StateRunning {
			continue
		}
		out = append(out, engine.Running{Name: name, Lifetime: v.recordLocked().Lifetime, PID: v.holder(), Guest: v.guest == engine.GuestConnected})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (d *Engine) Tell(machine, finding, text string) {
	// invariant: told once, to the open surfaces and to the owner's
	// notifications, as every word the engine says by itself; a disk under
	// its floor is the event of SPEC §10.4, any other finding the activity's.
	kind := engine.NoticeActivity
	if finding == activity.SubjectDisk {
		kind = engine.EventDiskLow
	}
	d.hub.tell(engine.Notice{Kind: kind, Text: text})
	d.notify(context.Background(), engine.Notification{Kind: kind, Subject: machine, Text: text})
}
