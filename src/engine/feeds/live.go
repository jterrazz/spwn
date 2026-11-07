package feeds

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/src/activity"
	activitycalls "github.com/jterrazz/studio-station/src/activity/calls"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/engine"
	pagemenubar "github.com/jterrazz/studio-station/src/pages/menubar"
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
	Calls   call.Router
	Clock   Clock
	ThisMac string
	// value: how long a tick waits on a late machine; zero is a second.
	Budget time.Duration

	mu sync.Mutex
	of map[string]*watched
}

type watched struct {
	readings []activity.Reading
	answered time.Time
	asking   chan struct{}
}

// invariant: a tick asks each machine its keys name for one reading, all
// at once, through its engine, and waits on them for its budget alone: a
// machine still late keeps its last value, stale once it has not answered
// for 30 s, and is asked again only once its answer came.
func (l *Live) Sample(ctx context.Context, keys []string) engine.Sample {
	var machines []string
	for _, k := range keys {
		if m := pagemenubar.LiveMachine(k); m != "" && !slices.Contains(machines, m) {
			machines = append(machines, m)
		}
	}
	waits := l.ask(ctx, machines)
	budget := l.Budget
	if budget == 0 {
		budget = tickBudget
	}
	timer := time.NewTimer(budget)
	defer timer.Stop()
wait:
	for _, done := range waits {
		select {
		case <-done:
		case <-timer.C:
			break wait
		case <-ctx.Done():
			return engine.Sample{}
		}
	}
	return l.compose(keys)
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
	var waits []chan struct{}
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

func (l *Live) compose(keys []string) engine.Sample {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.Clock.Now()
	values := pagemenubar.LiveValues(keys, func(m string) []activity.Reading {
		if w := l.of[m]; w != nil {
			return w.readings
		}
		return nil
	})
	out := engine.Sample{Values: map[string]any{}}
	for _, k := range keys {
		v, ok := values[k]
		if !ok {
			continue
		}
		if w := l.of[pagemenubar.LiveMachine(k)]; w != nil && now.Sub(w.answered) > freshFor {
			v.Stale = true
			out.Stale = append(out.Stale, k)
		}
		out.Values[k] = v
	}
	return out
}

// value: the readings of a machine the ticks kept, oldest first, so a page
// rebuilt for its content starts from the values on screen.
func (l *Live) Readings(name string) []activity.Reading {
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
