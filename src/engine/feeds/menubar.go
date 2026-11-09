package feeds

import (
	"context"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/src/awake"
	awakecalls "github.com/jterrazz/studio-station/src/awake/calls"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/engine"
	"github.com/jterrazz/studio-station/src/machines"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	"github.com/jterrazz/studio-station/src/overview"
	overviewcalls "github.com/jterrazz/studio-station/src/overview/calls"
	"github.com/jterrazz/studio-station/src/page"
	pagemenubar "github.com/jterrazz/studio-station/src/pages/menubar"
	"github.com/jterrazz/studio-station/src/ports"
	portscalls "github.com/jterrazz/studio-station/src/ports/calls"
)

const (
	sampleEvery = 2 * time.Second
	remoteEvery = 30 * time.Second
	todoEvery   = 15 * time.Minute
	sampleKept  = 16
)

type Menubar struct {
	// value: the router the feed reads this Mac through, its calls run in
	// the engine's process.
	Calls       call.Router
	Clock       Clock
	ThisMachine page.Context
	// value: the Power report of a view the engine holds.
	Report func(awake.View) *awake.Report
	// value: every Mac's Awake, This Mac's first, as the CLI's station awake
	// reads it, for the Awake page.
	EveryMac EveryMac
	// value: the readings the live ticks keep, which a page rebuilt for its
	// content starts from; nil starts each page pending.
	Live *Live
}

type Clock interface {
	Sleep(ctx context.Context, d time.Duration) error
	Now() time.Time
}

type menubar struct {
	o Menubar
	// value: the page the feed draws from its reading: home, or Find.
	draw func(pagemenubar.Reading) page.Page
	mu   sync.Mutex
	r    pagemenubar.Reading
	out  chan any
}

func (o Menubar) Feed() engine.Feed {
	return o.feed(func(r pagemenubar.Reading) page.Page {
		r.Asks = levels
		return pagemenubar.Page(r, o.ThisMachine)
	})
}

func (o Menubar) Find() engine.Feed {
	// invariant: SPEC B5.6: Find is drawn from home's own reading, read as
	// home reads it, so it asks nothing home does not.
	return o.feed(func(r pagemenubar.Reading) page.Page {
		return pagemenubar.Find(pagemenubar.FindReading{Reading: r, Now: o.Clock.Now(), Zone: time.Local, Asks: levels}, o.ThisMachine)
	})
}

func (o Menubar) feed(draw func(pagemenubar.Reading) page.Page) engine.Feed {
	return func(ctx context.Context, in engine.FeedInput) <-chan any {
		f := &menubar{o: o, draw: draw, out: make(chan any, 1)}
		go f.run(ctx, in)
		return f.out
	}
}

// invariant: the frame goes out at once, then a page each time one of the
// reads answers; every read runs only while this feed does, so nothing of it
// runs at rest, and the channel closes when the watch ends.
func (f *menubar) run(ctx context.Context, in engine.FeedInput) {
	defer close(f.out)
	f.update(ctx, func(r *pagemenubar.Reading) { r.Awake = f.o.Report(in.Awake()) })
	loops := []func(context.Context, engine.FeedInput){f.todo, f.remote, f.local}
	each := make([]chan struct{}, len(loops))
	for i := range each {
		each[i] = make(chan struct{}, 1)
	}
	go fanOut(ctx, in.Refresh, each)
	var wg sync.WaitGroup
	for i, loop := range loops {
		own := in
		own.Refresh = each[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			loop(ctx, own)
		}()
	}
	wg.Wait()
}

// invariant: a refresh of Home wakes every read of it, so what a call
// changed (a VM created, an update applied) is on the page once it returns.
func fanOut(ctx context.Context, from <-chan struct{}, to []chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-from:
			for _, c := range to {
				select {
				case c <- struct{}{}:
				default:
				}
			}
		}
	}
}

// invariant: Home is rebuilt when one of its reads answers, never on a
// tick: its moving numbers are the live keys it lists, and a rebuild starts
// from the readings the ticks kept.
func (f *menubar) update(ctx context.Context, change func(*pagemenubar.Reading)) {
	f.mu.Lock()
	change(&f.r)
	f.r.Now, f.r.Zone = f.o.Clock.Now(), time.Local
	f.r.Samples = f.o.Live.Readings(f.o.ThisMachine.Machine)
	p := page.Document{Schema: page.Schema, Page: f.draw(f.r)}
	f.mu.Unlock()
	select {
	case <-f.out:
	default:
	}
	select {
	case f.out <- p:
	case <-ctx.Done():
	}
}

// invariant: the To do is read whole at the feed's start, then again from
// fresh readings every 15 min or on a refresh.
func (f *menubar) todo(ctx context.Context, in engine.FeedInput) {
	fresh := false
	for {
		if snaps, err := call.Follow(ctx, f.o.Calls, machine.Ref{}, overviewcalls.Read, overviewcalls.ReadIn{Fresh: fresh}, nil); err == nil {
			for u := range snaps {
				if u.Err != nil {
					continue
				}
				summary := u.Value.Summary
				f.update(ctx, func(r *pagemenubar.Reading) { r.Home = &summary })
			}
		}
		if rest(ctx, f.o.Clock, todoEvery, engine.FeedInput{Refresh: in.Refresh}) != nil {
			return
		}
		fresh = true
	}
}

func (f *menubar) read(ctx context.Context, thisMac bool) (allMachines machines.AllMachines, portsRead ports.AllMachines, allMachinesErr, portsErr error) {
	listed, allMachinesErr := call.Read(ctx, f.o.Calls, machine.Ref{}, machinescalls.List, machinescalls.ListIn{ThisMac: thisMac, Unwritten: true}, nil)
	shared, portsErr := call.Invoke(ctx, f.o.Calls, machine.Ref{}, portscalls.List, portscalls.ListIn{AllMachines: !thisMac}, nil)
	return listed.All, shared.AllMachines, allMachinesErr, portsErr
}

func (f *menubar) remote(ctx context.Context, in engine.FeedInput) {
	for {
		fl, sv, ferr, serr := f.read(ctx, false)
		macs, away := f.awake(ctx, fl), f.away(ctx, fl)
		shared := f.o.everyMacAccounts(ctx, fl, "")
		f.update(ctx, func(r *pagemenubar.Reading) {
			if ferr == nil {
				r.All = &fl
				r.Macs = macs
				r.Away = away
				r.Accounts = &shared
			}
			if serr == nil {
				r.Ports = &sv
			}
		})
		if rest(ctx, f.o.Clock, remoteEvery, engine.FeedInput{Refresh: in.Refresh}) != nil {
			return
		}
	}
}

// value: each other Mac's Awake, as its own engine reports it; a Mac that
// does not answer is left out.
func (f *menubar) awake(ctx context.Context, fl machines.AllMachines) []awake.Report {
	var out []awake.Report
	for _, h := range fl.Macs {
		if h.ThisMac || !h.Reachable {
			continue
		}
		if r, err := call.Read(ctx, f.o.Calls, machine.Ref{Name: h.Name}, awakecalls.Read, awakecalls.ReadIn{}, nil); err == nil {
			r.Mac = h.Name
			out = append(out, r)
		}
	}
	return out
}

// value: each other Mac's Overview, as its own engine last read it; a Mac
// that does not answer is left out.
func (f *menubar) away(ctx context.Context, fl machines.AllMachines) []overview.Summary {
	var out []overview.Summary
	for _, h := range fl.Macs {
		if h.ThisMac || !h.Reachable {
			continue
		}
		if r, err := call.Read(ctx, f.o.Calls, machine.Ref{Name: h.Name}, overviewcalls.Read, overviewcalls.ReadIn{}, nil); err == nil {
			out = append(out, r.Summary)
		}
	}
	return out
}

// invariant: a change of the engine's own state rereads this host only and
// keeps the remote hosts as last read.
func (f *menubar) local(ctx context.Context, in engine.FeedInput) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-in.Changed:
		case <-in.Refresh:
		}
		fl, sv, ferr, serr := f.read(ctx, true)
		f.update(ctx, func(r *pagemenubar.Reading) {
			r.Awake = f.o.Report(in.Awake())
			if ferr == nil && len(fl.Macs) == 1 {
				r.All = hereAllMachines(r.All, fl)
			}
			if serr == nil && len(sv.Macs) == 1 {
				r.Ports = thisMacPorts(r.Ports, sv)
			}
		})
	}
}

// value: the fleet as last read with this host's row replaced by here's
// one; here itself when nothing was read yet.
func hereAllMachines(was *machines.AllMachines, thisMac machines.AllMachines) *machines.AllMachines {
	if was == nil {
		return &thisMac
	}
	out := *was
	out.Macs = append([]machines.MacView(nil), was.Macs...)
	for i, h := range out.Macs {
		if h.ThisMac {
			out.Macs[i] = thisMac.Macs[0]
		}
	}
	return &out
}

func thisMacPorts(was *ports.AllMachines, thisMac ports.AllMachines) *ports.AllMachines {
	if was == nil {
		return &thisMac
	}
	out := *was
	out.Macs = append([]ports.MacPorts(nil), was.Macs...)
	for i, h := range out.Macs {
		if h.ThisMac {
			out.Macs[i] = thisMac.Macs[0]
		}
	}
	return &out
}
