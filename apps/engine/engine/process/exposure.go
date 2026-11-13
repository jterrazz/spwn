package process

import (
	"context"
	"maps"
	"slices"
	"sort"
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

const (
	exposureEvery = 5 * time.Minute
)

func (d *Engine) ExposureKept() engine.ExposureKept {
	d.mu.Lock()
	defer d.mu.Unlock()
	return engine.ExposureKept{At: d.exposedAt, Scanned: maps.Clone(d.scanned), Red: slices.Sorted(slices.Values(d.red))}
}

func (d *Engine) RememberExposure(k engine.ExposureKept) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.exposedAt, d.scanned, d.red = k.At, maps.Clone(k.Scanned), slices.Clone(k.Red)
}

func (d *Engine) hear(vm string, ls []ports.Listener) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.heard[vm] = ls
	delete(d.lastHeard, vm)
}

// value: whether a guest reported listeners other than the job last read;
// the caller holds d.mu.
func (d *Engine) reheardLocked() bool {
	for vm, ls := range d.heard {
		if !slices.Equal(d.scanned[vm], ls) {
			return true
		}
	}
	return false
}

// invariant: a listener its guest marks quiet (in the guest's ephemeral
// range) is kept here, neither heard nor judged, so never shared by itself,
// until an owner's reservation names its port: then the next tick hears it
// as any other.
func (d *Engine) reserved(vm string, port int) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.reservationIndex(vm, port) >= 0
}

func (d *Engine) setQuiet(vm string, ls map[int]ports.Listener) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.quiet[vm] = ls
}

func (d *Engine) keepQuiet(vm string, l ports.Listener) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.quiet[vm] == nil {
		d.quiet[vm] = map[int]ports.Listener{}
	}
	d.quiet[vm][l.Port] = l
}

func (d *Engine) dropQuiet(vm string, port int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.quiet[vm], port)
}

func (d *Engine) hearReserved(ctx context.Context) {
	type found struct {
		vm string
		l  ports.Listener
	}
	var now []found
	d.mu.Lock()
	for _, r := range d.reservations {
		if l, ok := d.quiet[r.VM][r.GuestPort]; ok {
			now = append(now, found{r.VM, l})
			delete(d.quiet[r.VM], r.GuestPort)
		}
	}
	d.mu.Unlock()
	for _, f := range now {
		if heard := d.heardOf(f.vm); !slices.ContainsFunc(heard, func(h ports.Listener) bool { return h.Port == f.l.Port }) {
			d.hear(f.vm, append(heard, f.l))
		}
		d.listen(ctx, f.vm, f.l.Port, f.l.Bind, f.l.Process)
	}
}

func (d *Engine) heardOf(vm string) []ports.Listener {
	d.mu.Lock()
	defer d.mu.Unlock()
	return slices.Clone(d.heard[vm])
}

const restartingDetail = "restarting"

// value: why a VM last heard listening does not run its ports now; the
// caller holds d.mu.
func (d *Engine) haltLocked(v *vmState) ports.Halt {
	if v == nil {
		return ports.StoppedOnPurpose
	}
	rec := v.recordLocked()
	vm := model.VM{Lifetime: rec.Lifetime, State: model.VMState(v.state), Want: rec.Want, Detail: v.detail, Restarts: v.restarts}
	switch {
	case v.restarts.Looping:
		return ports.Down
	case v.state == engine.StateRunning, v.detail == restartingDetail:
		return ports.Restarting
	case model.Down(vm):
		return ports.Down
	}
	return ports.StoppedOnPurpose
}

// value: what listens in each running VM, placed behind its Mac or bridged
// onto its network, and what a VM down or booting again was last heard
// listening on, told it does not answer; the caller holds d.mu.
func (d *Engine) listeningLocked() []ports.Machine {
	names := make([]string, 0, len(d.heard)+len(d.lastHeard))
	for vm := range d.heard {
		names = append(names, vm)
	}
	for vm := range d.lastHeard {
		if _, heard := d.heard[vm]; !heard {
			names = append(names, vm)
		}
	}
	sort.Strings(names)
	out := make([]ports.Machine, 0, len(names))
	for _, vm := range names {
		v := d.vms[vm]
		ls, answer := d.heard[vm], ports.Answers
		if _, heard := d.heard[vm]; !heard {
			var listed bool
			if answer, listed = ports.AnswerOf(d.haltLocked(v)); !listed {
				continue
			}
			ls = d.lastHeard[vm]
		}
		m := ports.VMOf(vm, d.m.ThisMacName(), v != nil && v.recordLocked().Network == model.Bridged, slices.Clone(ls))
		m.Answer = answer
		out = append(out, m)
	}
	return out
}

// value: this Mac's ports as the engine serves them, with what listens in
// its VMs (in one VM when vm names it) in its class, declarations read, so a
// VM's own Overview counts its ports as its Mac's Ports page does.
func (d *Engine) surveyed(ctx context.Context, vm string) ports.MacPorts {
	out := d.macPorts(vm)
	d.mu.Lock()
	heard := d.listeningLocked()
	d.mu.Unlock()
	if vm != "" {
		heard = slices.DeleteFunc(heard, func(m ports.Machine) bool { return m.Ref.Name != vm })
	}
	var decl []ports.Declaration
	if d.o.Declarations != nil {
		decl, _ = d.o.Declarations(ctx)
	}
	out.Ports = ports.Survey(heard, decl)
	return out
}

// invariant: the exposure job (SPEC 5.6.1): this Mac's listeners and its
// VMs' are read at the start, every exposureEvery and on every listener
// change a guest reports; a port that turns red is told once, and again
// only after it stopped being red.
func (d *Engine) watchExposure(ctx context.Context) {
	if d.o.Listening == nil {
		return
	}
	now := d.h.Now()
	d.mu.Lock()
	if !d.reheardLocked() && !d.exposedAt.IsZero() && now.Sub(d.exposedAt) < exposureEvery {
		d.mu.Unlock()
		return
	}
	d.exposedAt, d.scanned = now, maps.Clone(d.heard)
	vms := d.listeningLocked()
	d.mu.Unlock()
	mine, _ := d.o.Listening(ctx)
	var decl []ports.Declaration
	if d.o.Declarations != nil {
		decl, _ = d.o.Declarations(ctx)
	}
	found := ports.Exposed(ports.Survey(append([]ports.Machine{ports.MacOf(d.m.ThisMacName(), mine)}, vms...), decl))
	d.mu.Lock()
	was := d.red
	d.red = make([]string, 0, len(found))
	var fresh []string
	for _, f := range found {
		d.red = append(d.red, f.Subject)
		if !slices.Contains(was, f.Subject) {
			fresh = append(fresh, ports.Told(f))
		}
	}
	d.mu.Unlock()
	for _, text := range fresh {
		d.tell(ctx, d.m.ThisMacName(), engine.Told{Kind: engine.EventExposed, Parts: []string{text}})
	}
}
