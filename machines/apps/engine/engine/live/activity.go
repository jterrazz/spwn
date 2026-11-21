package live

import (
	"context"
	"strings"
	"time"

	activitycalls "github.com/jterrazz/studio-station/apps/engine/activity/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageactivity "github.com/jterrazz/studio-station/apps/engine/pages/activity"
	pagemenubar "github.com/jterrazz/studio-station/apps/engine/pages/menubar"
)

type Activity struct {
	// value: the router the chart is read through: This Mac's in the
	// engine's process, a VM's and another Mac's through their engine.
	Calls       call.Router
	Clock       Clock
	ThisMachine page.Context
	// value: the rings that know the Mac of each VM; nil reads a name as a
	// Mac's.
	Live *Live
}

func (a Activity) Page(ctx context.Context, subject string) <-chan any {
	// invariant: SPEC §10.2 Activity, §6.3, B5.16: a machine's measures over
	// the window the page names after its colon (1 h unless named), This
	// Mac's unless a machine is named before the window
	// (`menubar.activity:infra:6m`), read every 2 s while the page is open,
	// each read a page; the channel closes when the watch ends.
	out := make(chan any, 1)
	name, window := activitySubject(subject)
	go func() {
		defer close(out)
		for {
			p := a.read(ctx, name, window)
			if ctx.Err() != nil {
				return
			}
			select {
			case <-out:
			default:
			}
			select {
			case out <- page.Document{Schema: page.Schema, Page: p}:
			case <-ctx.Done():
				return
			}
			if a.Clock.Sleep(ctx, sampleEvery) != nil {
				return
			}
		}
	}()
	return out
}

const sampleEvery = 2 * time.Second

func (a Activity) read(ctx context.Context, name string, window time.Duration) page.Page {
	at, v, ask := a.of(ctx, name)
	v.Graph, ask.Graph = window, window
	snaps, err := call.Follow(ctx, a.Calls, at, activitycalls.Read, ask, nil)
	if err != nil {
		return pagemenubar.Unread(pagemenubar.ActivityID, a.ThisMachine, err)
	}
	for u := range snaps {
		if u.Err != nil {
			return pagemenubar.Unread(pagemenubar.ActivityID, a.ThisMachine, u.Err)
		}
		v.Report = u.Value
	}
	return pagemenubar.Activity(v, a.ThisMachine)
}

// value: the machine and the window an Activity page names: a window alone
// is This Mac's, a machine alone is its last hour.
func activitySubject(subject string) (name string, window time.Duration) {
	name, words, named := strings.Cut(subject, ":")
	if !named {
		words = subject
	}
	if d, err := time.ParseDuration(words); err == nil {
		window = d
		if !named {
			name = ""
		}
	}
	if window <= 0 {
		window = time.Hour
	}
	return name, window
}

// value: SPEC B5.16: where a machine's activity is read, through its
// engine: a Mac on itself, a VM on the Mac that runs it, whose engine keeps
// its guest's history.
func (a Activity) of(ctx context.Context, name string) (machine.Ref, pageactivity.View, activitycalls.ReadIn) {
	if name == "" || name == a.ThisMachine.Machine {
		return machine.Ref{}, pageactivity.View{Name: a.ThisMachine.Machine, Label: "This Mac", Kind: string(machine.Computer), ThisMac: true}, activitycalls.ReadIn{}
	}
	mac, isVM := "", false
	if a.Live != nil {
		mac, isVM = a.Live.macOf(ctx, name)
	}
	if !isVM {
		return machine.Ref{Name: name}, pageactivity.View{Name: name, Label: name, Kind: string(machine.Computer)}, activitycalls.ReadIn{}
	}
	at := machine.Ref{Name: mac}
	if mac == a.ThisMachine.Machine {
		at = machine.Ref{}
	}
	v := pageactivity.View{Name: name, Label: name, Mac: mac, On: mac}
	return at, v, activitycalls.ReadIn{Of: name, Machine: machine.Ref{Name: name, On: mac}}
}
