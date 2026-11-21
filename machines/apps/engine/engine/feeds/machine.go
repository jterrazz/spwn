package feeds

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	activitycalls "github.com/jterrazz/studio-station/apps/engine/activity/calls"
	"github.com/jterrazz/studio-station/apps/engine/awake"
	awakecalls "github.com/jterrazz/studio-station/apps/engine/awake/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
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
	// the channel closes when the watch ends. Its first frame is drawn from
	// the machines Home last read, when they hold the machine.
	return func(ctx context.Context, in engine.FeedInput) <-chan any {
		out := make(chan any, 1)
		m := &one{o: o, out: out, m: pagemenubar.Machine{Name: in.Subject, Asks: levels, Zone: o.zone(), All: holding(o.HomeRead.Last(), in.Subject)}}
		go func() {
			defer close(out)
			m.send(func(*pagemenubar.Machine) {})
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

// invariant: the reads answer at once, so the page they draw replaces the
// one not yet taken under the same lock it is drawn under: the page waiting
// is always the newest, and a send never waits on the reader.
func (f *one) send(change func(*pagemenubar.Machine)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	change(&f.m)
	f.m.Now = f.o.Clock.Now()
	f.m.Samples = f.o.Live.Readings(f.m.Name)
	p := page.Document{Schema: page.Schema, Page: pagemenubar.MachinePage(f.m)}
	select {
	case <-f.out:
	default:
	}
	select {
	case f.out <- p:
	default:
	}
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
	for {
		f.cycle(ctx)
		if rest(ctx, f.o.Clock, remoteEvery, in) != nil {
			return
		}
	}
}

// invariant: a cycle asks every read of the page at once and draws each
// answer as it comes, so the page is whole once its slowest read answered,
// and a Mac that does not answer holds back its own parts alone. The reads
// that need the machines start on the machines the page already draws: the
// Awake, or else on this cycle's first rows that hold the machine; the
// sign-ins, of every Mac, or else on every Mac this cycle read.
func (f *one) cycle(ctx context.Context) {
	f.mu.Lock()
	drawn := f.m.All
	name := f.m.Name
	f.mu.Unlock()
	first := make(chan machines.AllMachines, 1)
	if drawn != nil {
		first <- *drawn
	}
	var whole *machines.AllMachines
	listed := make(chan struct{})
	at := machine.Ref{Name: name}
	var wg sync.WaitGroup
	run := func(read func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			read()
		}()
	}
	run(func() {
		defer close(listed)
		whole = f.allMachines(ctx, name, first)
	})
	run(func() {
		shared, err := call.Invoke(ctx, f.o.Calls, machine.Ref{}, portscalls.List, portscalls.ListIn{AllMachines: true}, nil)
		if err == nil {
			f.send(func(m *pagemenubar.Machine) { m.Ports = &shared.AllMachines })
		}
	})
	if name == f.o.ThisMachine.Machine {
		run(func() {
			own, err := call.Invoke(ctx, f.o.Calls, machine.Ref{}, portscalls.List, portscalls.ListIn{}, nil)
			if err == nil && len(own.AllMachines.Macs) == 1 {
				f.send(func(m *pagemenubar.Machine) { m.Ports = thisMacPorts(m.Ports, own.AllMachines) })
			}
		})
	}
	run(func() {
		read, err := call.Read(ctx, f.o.Calls, at, overviewcalls.Read, overviewcalls.ReadIn{}, nil)
		if err == nil {
			f.send(func(m *pagemenubar.Machine) { m.Summary = &read.Summary })
		}
	})
	run(func() {
		kept, err := call.Read(ctx, f.o.Calls, at, activitycalls.Read, activitycalls.ReadIn{History: true, Graph: 6 * time.Minute, Quick: true}, nil)
		f.send(func(m *pagemenubar.Machine) { m.Activity = keptAfter(m.Activity, kept, err) })
	})
	run(func() {
		select {
		case all := <-first:
			f.awake(ctx, name, all)
		case <-listed:
			select {
			case all := <-first:
				f.awake(ctx, name, all)
			default:
			}
		case <-ctx.Done():
		}
	})
	run(func() {
		all := drawn
		if all == nil {
			select {
			case <-listed:
				all = whole
			case <-ctx.Done():
			}
		}
		if all != nil {
			f.signIns(ctx, name, *all)
		}
	})
	wg.Wait()
}

// invariant: the machines are drawn as they fill: each Mac's row as its answer
// comes, over the rows the page already draws, then the last answer whole,
// which it returns.
func (f *one) allMachines(ctx context.Context, name string, first chan<- machines.AllMachines) *machines.AllMachines {
	updates, err := call.Follow(ctx, f.o.Calls, machine.Ref{}, machinescalls.List, machinescalls.ListIn{Unwritten: true}, nil)
	if err != nil {
		return nil
	}
	var last *machines.AllMachines
	for u := range updates {
		if u.Err != nil {
			last = nil
			continue
		}
		all := u.Value.All
		last = &all
		if holding(&all, name) == nil {
			continue
		}
		select {
		case first <- all:
		default:
		}
		f.send(func(m *pagemenubar.Machine) { m.All = rowsOver(m.All, all) })
	}
	if last != nil {
		f.send(func(m *pagemenubar.Machine) { m.All = last })
	}
	return last
}

// value: the Mac a page's machine is on: itself, or the Mac that runs it.
func macOf(all machines.AllMachines, name string) (machines.MacView, bool) {
	if h, ok := macNamed(all, name); ok {
		return h, true
	}
	h, _ := macOfVM(all, name)
	return h, false
}

func (f *one) awake(ctx context.Context, name string, all machines.AllMachines) {
	h, _ := macOf(all, name)
	if h.Name == "" {
		return
	}
	var held *awake.Report
	if r, err := call.Read(ctx, f.o.Calls, macRef(h), awakecalls.Read, awakecalls.ReadIn{}, nil); err == nil {
		r.Mac = h.Name
		held = &r
	}
	f.send(func(m *pagemenubar.Machine) { m.Awake = held })
}

// invariant: a Mac's page reads every Mac's sign-ins, a VM's those of
// its Mac.
func (f *one) signIns(ctx context.Context, name string, all machines.AllMachines) {
	h, isMac := macOf(all, name)
	if h.Name == "" {
		return
	}
	only := h.Name
	if isMac {
		only = ""
	}
	signed := f.o.signIns(ctx, all, only)
	f.send(func(m *pagemenubar.Machine) { m.SignIns = &signed })
}

// value: the machines when they hold the machine, a Mac or a VM; nil
// otherwise.
func holding(all *machines.AllMachines, name string) *machines.AllMachines {
	if all == nil {
		return nil
	}
	if _, ok := macNamed(*all, name); ok {
		return all
	}
	if _, ok := macOfVM(*all, name); ok {
		return all
	}
	return nil
}

// value: the machines as drawn with each Mac of rows in place of its row;
// rows itself when nothing was drawn yet.
func rowsOver(was *machines.AllMachines, rows machines.AllMachines) *machines.AllMachines {
	if was == nil {
		return &rows
	}
	out := *was
	out.Macs = slices.Clone(was.Macs)
	for _, r := range rows.Macs {
		i := slices.IndexFunc(out.Macs, func(h machines.MacView) bool { return h.Name == r.Name })
		if i < 0 {
			out.Macs = append(out.Macs, r)
			continue
		}
		out.Macs[i] = r
	}
	return &out
}

func keptAfter(was *activity.Report, read activity.Report, err error) *activity.Report {
	// invariant: a read that fails keeps the last values on the page, except
	// a machine that does not answer: it keeps none, its Activity dropped.
	var failed *fail.Error
	switch {
	case err == nil:
		return &read
	case errors.As(err, &failed) && failed.Type == fail.Unreachable:
		return nil
	}
	return was
}
