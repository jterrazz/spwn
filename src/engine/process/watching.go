package process

import (
	"context"
	"slices"
	"sync"

	"github.com/jterrazz/studio-station/src/engine"
)

type watchers struct {
	mu sync.Mutex
	on map[string]int
}

func (w *watchers) hold(machines []string, by int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.on == nil {
		w.on = map[string]int{}
	}
	for _, m := range machines {
		w.on[m] += by
		if w.on[m] <= 0 {
			delete(w.on, m)
		}
	}
}

func (w *watchers) watched(machine string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.on[machine] > 0
}

type Watching struct {
	Machines []string `json:"machines"`
}

func (d *Engine) Watch(ctx context.Context, machines []string, send func(any) error) error {
	// invariant: SPEC §6.3: a machine is watched while a surface holds this
	// watch open on it (the menu bar open), and only then; a watch of pages or
	// events alone, which the menu bar holds from its launch, never makes one.
	held := slices.Compact(slices.Sorted(slices.Values(machines)))
	if len(held) == 0 {
		held = []string{engine.ThisMac}
	}
	d.watchers.hold(held, 1)
	defer d.watchers.hold(held, -1)
	if err := send(Watching{Machines: slices.DeleteFunc(slices.Clone(held), func(m string) bool { return m == engine.ThisMac })}); err != nil {
		return nil
	}
	<-ctx.Done()
	return nil
}

func (d *Engine) Watched(machine string) bool { return d.watchers.watched(machine) }
