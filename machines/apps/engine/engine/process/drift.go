package process

import (
	"context"
	"slices"
	"sort"
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"

	"github.com/jterrazz/studio-station/apps/engine/engine"
)

const (
	// value: how long a workspace's reading of its file holds before the
	// drift job reads it again, and how long one reading may take.
	driftEvery = 10 * time.Minute
	driftBound = 2 * time.Minute
)

// invariant: the drift job: each running workspace whose guest is
// connected is read once after each boot, every driftEvery, and again once
// a call of Options.Changes ended on it; one reading per VM at a time, run
// beside the loop, so no watch and no step waits on it. A server, which
// station sets up nothing inside, is never read.
func (d *Engine) keepDrift(ctx context.Context) {
	if d.o.Drift == nil {
		return
	}
	now := d.h.Now()
	var due []string
	d.locked(func() {
		for name, v := range d.vms {
			if v.state != engine.StateRunning || v.client == nil || v.drifting || v.recordLocked().Lifetime == model.AlwaysOn {
				continue
			}
			if kept := v.keptDriftLocked(); kept != nil && !v.driftDue && now.Sub(kept.At) < driftEvery {
				continue
			}
			v.drifting, v.driftDue = true, false
			due = append(due, name)
		}
	})
	sort.Strings(due)
	beside := d.o.Beside
	if beside == nil {
		beside = func(f func()) { go f() }
	}
	for _, name := range due {
		beside(func() {
			bound, cancel := context.WithTimeout(ctx, driftBound)
			read := d.o.Drift(bound, name)
			cancel()
			read.At = now
			d.mu.Lock()
			defer d.mu.Unlock()
			if v := d.vms[name]; v != nil {
				v.drift, v.drifting = &read, false
			}
		})
	}
}

// value: the reading a VM's head says, while it runs and since its boot;
// nil before it. The caller holds d.mu.
func (v *vmState) keptDriftLocked() *model.KeptDrift {
	if v.drift == nil || v.state != engine.StateRunning || v.drift.At.Before(v.bootAt) {
		return nil
	}
	kept := *v.drift
	return &kept
}

// invariant: a call of Options.Changes that ended on a VM makes its
// reading due at the next step, whatever it answered.
func (d *Engine) changed(vm, call string) {
	if call == "" || !slices.Contains(d.o.Changes, call) {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if v := d.vms[vm]; v != nil {
		v.driftDue = true
	}
}
