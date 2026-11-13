package feeds

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	activitycalls "github.com/jterrazz/studio-station/apps/engine/activity/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	pagemenubar "github.com/jterrazz/studio-station/apps/engine/pages/menubar"
)

const (
	// value: how long a machine may go unheard while a page reads it before
	// its values turn stale (SPEC §B5.15).
	freshFor = 30 * time.Second
	// value: how long one tick waits on the machines it asked before it
	// goes out with the last value of those still late.
	tickBudget = time.Second
)

type Live struct {
	// value: the router each machine's sample rides: This Mac's in the
	// engine's process, a VM's and another Mac's through their engine.
	Calls call.Router
	// value: the Mac that runs a VM, by the VM's name; ok false for a name
	// that is no VM, or that no Mac answers for yet.
	MacOf   func(ctx context.Context, vm string) (mac string, ok bool)
	Clock   Clock
	ThisMac string
	// value: how long a tick waits on a late machine; zero is a second.
	Budget time.Duration

	mu   sync.Mutex
	of   map[string]*watched
	macs map[string]string
}

type watched struct {
	readings []activity.Reading
	answered time.Time
	asking   chan struct{}
}

func (l *Live) Sample(ctx context.Context, keys []string) engine.Sample {
	// invariant: a tick asks each machine its keys name for one reading, all
	// at once, through its engine, and waits on them for its budget alone: a
	// machine still late keeps its last value, stale once it has not answered
	// for 30 s, and is asked again only once its answer came.
	budget := l.Budget
	if budget == 0 {
		budget = tickBudget
	}
	spent, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	of := l.machinesOf(spent, keys)
	var machines []string
	for _, m := range of {
		if !slices.Contains(machines, m) {
			machines = append(machines, m)
		}
	}
	waits := l.ask(ctx, machines)
wait:
	for _, done := range waits {
		select {
		case <-done:
		case <-spent.Done():
			break wait
		}
	}
	if ctx.Err() != nil {
		return engine.Sample{}
	}
	return l.compose(keys, of)
}

// value: the machine each key is read on: its own, but a VM's share of its
// Mac (SPEC B5.3) on the Mac that runs it, which the tick finds once per VM;
// a key whose machine is not known yet is left out of the tick.
func (l *Live) machinesOf(ctx context.Context, keys []string) map[string]string {
	of := map[string]string{}
	for _, k := range keys {
		measure, m := engine.LiveOf(k)
		if m == "" {
			continue
		}
		if measure == engine.LiveCPUMac {
			mac, ok := l.macOf(ctx, m)
			if !ok {
				continue
			}
			m = mac
		}
		of[k] = m
	}
	return of
}

func (l *Live) macOf(ctx context.Context, vm string) (string, bool) {
	l.mu.Lock()
	mac, ok := l.macs[vm]
	l.mu.Unlock()
	if ok || l.MacOf == nil {
		return mac, ok
	}
	if mac, ok = l.MacOf(ctx, vm); !ok {
		return "", false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.macs == nil {
		l.macs = map[string]string{}
	}
	l.macs[vm] = mac
	return mac, true
}

func (l *Live) ask(ctx context.Context, machines []string) []chan struct{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.of == nil {
		l.of = map[string]*watched{}
	}
	for m, w := range l.of {
		if !slices.Contains(machines, m) && w.asking == nil {
			delete(l.of, m)
		}
	}
	waits := make([]chan struct{}, 0, len(machines))
	for _, m := range machines {
		w := l.of[m]
		if w == nil {
			w = &watched{}
			l.of[m] = w
		}
		if w.asking == nil {
			w.asking = make(chan struct{})
			go l.read(ctx, m, w)
		}
		waits = append(waits, w.asking)
	}
	return waits
}

func (l *Live) read(ctx context.Context, name string, w *watched) {
	at := machine.Ref{Name: name}
	if name == l.ThisMac {
		at = machine.Ref{}
	}
	ctx, cancel := context.WithTimeout(ctx, freshFor)
	defer cancel()
	r, err := call.Invoke(ctx, l.Calls, at, activitycalls.Sample, activitycalls.SampleIn{}, nil)
	l.mu.Lock()
	defer l.mu.Unlock()
	if err == nil {
		w.readings = append(w.readings, r)
		if len(w.readings) > sampleKept {
			w.readings = w.readings[len(w.readings)-sampleKept:]
		}
		w.answered = l.Clock.Now()
	}
	close(w.asking)
	w.asking = nil
}

func (l *Live) compose(keys []string, of map[string]string) engine.Sample {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.Clock.Now()
	out := engine.Sample{Values: map[string]any{}}
	for _, k := range keys {
		w := l.of[of[k]]
		if w == nil {
			continue
		}
		v := pagemenubar.LiveValue(k, w.readings)
		if v.Kind == "" {
			continue
		}
		if now.Sub(w.answered) > freshFor {
			v.Stale = true
			out.Stale = append(out.Stale, k)
		}
		out.Values[k] = v
	}
	return out
}

func (l *Live) Readings(name string) []activity.Reading {
	// value: the readings of a machine the ticks kept, oldest first, so a page
	// rebuilt for its content starts from the values on screen.
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if w := l.of[name]; w != nil {
		return slices.Clone(w.readings)
	}
	return nil
}
