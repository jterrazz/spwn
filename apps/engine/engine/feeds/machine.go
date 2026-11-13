package feeds

import (
	"context"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	accountscalls "github.com/jterrazz/studio-station/apps/engine/accounts/calls"
	activitycalls "github.com/jterrazz/studio-station/apps/engine/activity/calls"
	"github.com/jterrazz/studio-station/apps/engine/agents"
	agentscalls "github.com/jterrazz/studio-station/apps/engine/agents/calls"
	"github.com/jterrazz/studio-station/apps/engine/awake"
	awakecalls "github.com/jterrazz/studio-station/apps/engine/awake/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	overviewcalls "github.com/jterrazz/studio-station/apps/engine/overview/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagemenubar "github.com/jterrazz/studio-station/apps/engine/pages/menubar"
	portscalls "github.com/jterrazz/studio-station/apps/engine/ports/calls"
)

func (o Menubar) Machine() engine.Feed {
	// invariant: SPEC §10.3: a machine's page is read only while it is open:
	// every 30 s and again as soon as the engine's state moves or a refresh
	// names the page, each through the router, so another Mac's page is that
	// Mac's engine's answer; its activity is its live keys, never a rebuild;
	// the channel closes when the watch ends.
	return func(ctx context.Context, in engine.FeedInput) <-chan any {
		out := make(chan any, 1)
		m := &one{o: o, out: out, m: pagemenubar.Machine{Name: in.Subject, Asks: levels}}
		go func() {
			defer close(out)
			m.send(ctx, func(*pagemenubar.Machine) {})
			m.reads(ctx, in)
		}()
		return out
	}
}

// value: the wait between two reads of a page: its period, cut short by the
// engine's state moving or by a refresh; an error once the watch ends.
func rest(ctx context.Context, clock Clock, every time.Duration, in engine.FeedInput) error {
	wait, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		select {
		case <-in.Changed:
		case <-in.Refresh:
		case <-wait.Done():
		}
		cancel()
	}()
	_ = clock.Sleep(wait, every)
	return ctx.Err()
}

func latest(ctx context.Context, out chan any, p any) {
	select {
	case <-out:
	default:
	}
	select {
	case out <- p:
	case <-ctx.Done():
	}
}

// value: the confirmation rule of the calls, which the page only draws.
func levels(gesture string, kind machine.Kind, facts ...string) string {
	graded := make([]confirm.Fact, 0, len(facts))
	for _, f := range facts {
		graded = append(graded, confirm.Fact(f))
	}
	return string(confirm.Of(confirm.Action(gesture), kind, graded...))
}

type one struct {
	o   Menubar
	mu  sync.Mutex
	m   pagemenubar.Machine
	out chan any
}

func (f *one) send(ctx context.Context, change func(*pagemenubar.Machine)) {
	f.mu.Lock()
	change(&f.m)
	f.m.Now = f.o.Clock.Now()
	f.m.Samples = f.o.Live.Readings(f.m.Name)
	p := page.Document{Schema: page.Schema, Page: pagemenubar.MachinePage(f.m)}
	f.mu.Unlock()
	latest(ctx, f.out, p)
}

// value: the Mac that runs the VM a page is of.
func macOfVM(all machines.AllMachines, vm string) (machines.MacView, bool) {
	for _, h := range all.Macs {
		for _, v := range h.VMs {
			if v.Name == vm {
				return h, true
			}
		}
	}
	return machines.MacView{}, false
}

// value: the Mac a page is of, when it is a Mac's.
func macNamed(all machines.AllMachines, name string) (machines.MacView, bool) {
	for _, h := range all.Macs {
		if h.Name == name {
			return h, true
		}
	}
	return machines.MacView{}, false
}

func (f *one) reads(ctx context.Context, in engine.FeedInput) {
	at := machine.Ref{Name: f.m.Name}
	for {
		listed, ferr := call.Read(ctx, f.o.Calls, machine.Ref{}, machinescalls.List, machinescalls.ListIn{Unwritten: true}, nil)
		shared, serr := call.Invoke(ctx, f.o.Calls, machine.Ref{}, portscalls.List, portscalls.ListIn{AllMachines: true}, nil)
		shares, aerr := call.Read(ctx, f.o.Calls, machine.Ref{}, accountscalls.List, accountscalls.ListIn{VMs: true}, nil)
		read, oerr := call.Read(ctx, f.o.Calls, at, overviewcalls.Read, overviewcalls.ReadIn{}, nil)
		kept, kerr := call.Read(ctx, f.o.Calls, at, activitycalls.Read, activitycalls.ReadIn{History: true, Graph: 6 * time.Minute, Quick: true}, nil)
		var copies *accounts.AllMachines
		var held *awake.Report
		var hosted []agents.Agent
		h, isMac := macNamed(listed.All, f.m.Name)
		if ferr == nil && isMac {
			all := f.o.everyMacAccounts(ctx, listed.All, "")
			copies = &all
		}
		if ferr == nil && !isMac {
			h, _ = macOfVM(listed.All, f.m.Name)
			if h.Name != "" {
				all := f.o.everyMacAccounts(ctx, listed.All, h.Name)
				copies = &all
			}
		}
		if ferr == nil {
			if read, err := call.Invoke(ctx, f.o.Calls, machine.Ref{}, agentscalls.List, agentscalls.ListIn{}, nil); err == nil {
				hosted = append([]agents.Agent{}, read...)
			}
		}
		if ferr == nil && h.Name != "" {
			if r, err := call.Read(ctx, f.o.Calls, macRef(h), awakecalls.Read, awakecalls.ReadIn{}, nil); err == nil {
				r.Mac = h.Name
				held = &r
			}
		}
		f.send(ctx, func(m *pagemenubar.Machine) {
			m.Zone, m.Shares, m.Awake, m.Agents = time.Local, copies, held, hosted
			if ferr == nil {
				m.All = &listed.All
			}
			if serr == nil {
				m.Ports = &shared.AllMachines
			}
			if aerr == nil {
				m.Accounts = &shares
			}
			if oerr == nil {
				m.Summary = &read.Summary
			}
			if kerr == nil {
				m.Activity = &kept
			}
		})
		if rest(ctx, f.o.Clock, remoteEvery, in) != nil {
			return
		}
	}
}
