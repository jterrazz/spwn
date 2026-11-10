package calls

import (
	"context"
	"errors"
	"slices"

	accountscalls "github.com/jterrazz/studio-station/src/accounts/calls"
	activitycalls "github.com/jterrazz/studio-station/src/activity/calls"
	appscalls "github.com/jterrazz/studio-station/src/apps/calls"
	awakecalls "github.com/jterrazz/studio-station/src/awake/calls"
	backgroundcalls "github.com/jterrazz/studio-station/src/background/calls"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/finding"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/core/section"
	diskcalls "github.com/jterrazz/studio-station/src/disk/calls"
	"github.com/jterrazz/studio-station/src/engine"
	enginecalls "github.com/jterrazz/studio-station/src/engine/calls"
	"github.com/jterrazz/studio-station/src/machines"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	"github.com/jterrazz/studio-station/src/overview"
	portscalls "github.com/jterrazz/studio-station/src/ports/calls"
	settingscalls "github.com/jterrazz/studio-station/src/settings/calls"
)

type sectionRead func(ctx context.Context, r call.Router, self machine.Ref) (overview.Read, error)

// value: the read of each section an Overview has a line for, by its call
// (SSH is the door, by the machine's name).
func sectionReads() map[section.ID]sectionRead {
	thisMac := machine.Ref{}
	return map[section.ID]sectionRead{
		section.Activity: func(ctx context.Context, r call.Router, _ machine.Ref) (overview.Read, error) {
			out, err := call.Read(ctx, r, thisMac, activitycalls.Read, activitycalls.ReadIn{Quick: true}, nil)
			return overview.ActivityRead(out), err
		},
		section.Apps: func(ctx context.Context, r call.Router, self machine.Ref) (overview.Read, error) {
			out, err := call.Read(ctx, r, thisMac, appscalls.List, appscalls.ListIn{}, nil)
			return overview.AppsRead(out, self), err
		},
		section.Settings: func(ctx context.Context, r call.Router, self machine.Ref) (overview.Read, error) {
			out, err := call.Read(ctx, r, thisMac, settingscalls.List, settingscalls.ListIn{}, nil)
			return overview.SettingsRead(out, self), err
		},
		section.Disk: func(ctx context.Context, r call.Router, self machine.Ref) (overview.Read, error) {
			out, err := call.Read(ctx, r, thisMac, diskcalls.Read, diskcalls.ReadIn{}, nil)
			return overview.DiskRead(out, self), err
		},
		section.Ports: func(ctx context.Context, r call.Router, _ machine.Ref) (overview.Read, error) {
			out, err := call.Invoke(ctx, r, thisMac, portscalls.List, portscalls.ListIn{}, nil)
			return overview.PortsRead(out), err
		},
		section.Accounts: func(ctx context.Context, r call.Router, self machine.Ref) (overview.Read, error) {
			out, err := call.Read(ctx, r, thisMac, accountscalls.List, accountscalls.ListIn{}, nil)
			return overview.AccountsRead(out, self), err
		},
		section.Background: func(ctx context.Context, r call.Router, self machine.Ref) (overview.Read, error) {
			out, err := call.Invoke(ctx, r, thisMac, backgroundcalls.List, backgroundcalls.ListIn{}, nil)
			return overview.BackgroundRead(out, self), err
		},
		section.Awake: func(ctx context.Context, r call.Router, self machine.Ref) (overview.Read, error) {
			out, err := call.Read(ctx, r, thisMac, awakecalls.Read, awakecalls.ReadIn{}, nil)
			return overview.AwakeRead(out, self), err
		},
		section.SSH: func(_ context.Context, _ call.Router, self machine.Ref) (overview.Read, error) {
			return overview.SSHRead(self), nil
		},
	}
}

type answer struct {
	id        section.ID
	read      overview.Read
	err       error
	all       *machines.AllMachines
	readiness []overview.Check
}

func (r *Reader[App, Item, Misplaced]) Compose(ctx context.Context) <-chan overview.Summary {
	// invariant: overview.read owns no fact: every line is a section's own
	// read through the Router, in the one order of the machine's kind; the
	// first value has every line pending, each answer fills its own, and the
	// last is the whole Overview.
	kind := r.d.Kind
	head := overview.Head{Name: r.d.ThisMac.Hostname, Kind: kind, ThisMac: true}
	self := machine.Ref{Name: head.Name}
	if r.d.Meter != nil {
		if sample, err := r.d.Meter.Sample(ctx); err == nil && !sample.Boot.IsZero() {
			head.Up = sample.At.Sub(sample.Boot)
		}
	}
	reads := sectionReads()
	var ids []section.ID
	for _, id := range section.For(kind) {
		if _, ok := reads[id]; ok {
			ids = append(ids, id)
		}
	}
	var reading *engine.Reading
	if kind == machine.Mac {
		// invariant: the engine is read before the sections, so a section
		// that is the engine's reads it already answering, or says it waits.
		if out, err := call.Invoke(ctx, r.d.Calls, machine.Ref{}, enginecalls.Read, enginecalls.ReadIn{}, nil); err == nil {
			reading = &out
		}
	}
	answers := make(chan answer, len(ids)+1)
	for _, id := range ids {
		go func() {
			read, err := reads[id](ctx, r.d.Calls, self)
			answers <- answer{id: id, read: read, err: err}
		}()
	}
	left := len(ids)
	if kind == machine.Mac {
		left++
		go func() {
			out, err := call.Read(ctx, r.d.Calls, machine.Ref{}, machinescalls.List, machinescalls.ListIn{ThisMac: true, Unwritten: true}, nil)
			if err != nil {
				answers <- answer{err: err}
				return
			}
			answers <- answer{all: &out.All}
		}()
		left++
		go func() { answers <- answer{readiness: r.readiness(ctx)} }()
	}
	out := make(chan overview.Summary)
	go func() {
		defer close(out)
		got := map[section.ID]answer{}
		var all *machines.AllMachines
		var readiness []overview.Check
		send := func() bool {
			sum := summarize(head, ids, got, reading, all)
			sum.Readiness = readiness
			select {
			case out <- sum:
				return true
			case <-ctx.Done():
				return false
			}
		}
		if !send() {
			return
		}
		for ; left > 0; left-- {
			var a answer
			select {
			case a = <-answers:
			case <-ctx.Done():
				return
			}
			switch {
			case a.all != nil:
				all = a.all
			case a.readiness != nil:
				readiness = a.readiness
			case a.id != "":
				got[a.id] = a
			}
			if !send() {
				return
			}
		}
	}()
	return out
}

func summarize(head overview.Head, ids []section.ID, got map[section.ID]answer, reading *engine.Reading, all *machines.AllMachines) overview.Summary {
	state, own := engineOf(head, reading)
	var vms []overview.VM
	if all != nil {
		var found []finding.Finding
		found, vms = thisMacsVMs(*all)
		own = append(own, found...)
	}
	reads := make([]overview.Read, 0, len(ids))
	for _, id := range ids {
		a, ok := got[id]
		var refused *fail.Error
		switch {
		case !ok:
			reads = append(reads, overview.Pending(id))
		case errors.As(a.err, &refused) && refused.Type == fail.NotForThisMachine:
		case a.err != nil && state == overview.EngineStopped:
			reads = append(reads, overview.Unread(id, "needs the engine"))
		case a.err != nil:
			reads = append(reads, overview.Unread(id, "not read"))
		default:
			reads = append(reads, a.read)
		}
	}
	return overview.Summarize(head, state, own, reads, vms)
}

// value: a Mac's engine as engine.read tells it, and the finding a stopped
// one is; a machine whose engine is its Mac's has none of its own.
func engineOf(head overview.Head, r *engine.Reading) (overview.Engine, []finding.Finding) {
	if r == nil || r.State == "" {
		return overview.EngineNone, nil
	}
	mac := machine.Ref{Name: head.Name}
	switch r.State {
	case engine.EngineStopped:
		return overview.EngineStopped, []finding.Finding{{Machine: mac, Section: section.Overview, Severity: finding.NeedsALook, Subject: "engine",
			Text: "Engine stopped", Fix: finding.Fix{Call: "engine.start"}}}
	case engine.EngineNotInstalled:
		return overview.EngineStopped, []finding.Finding{{Machine: mac, Section: section.Overview, Severity: finding.NeedsALook, Subject: "engine",
			Text: "Engine not installed", Fix: finding.Fix{Call: "engine.start"}}}
	}
	return overview.EngineRunning, nil
}

// value: what a Mac's Overview reads of All machines: its VMs and their
// findings; the other Macs are All machines' own.
func thisMacsVMs(f machines.AllMachines) ([]finding.Finding, []overview.VM) {
	i := slices.IndexFunc(f.Macs, func(h machines.MacView) bool { return h.ThisMac })
	if i < 0 {
		return nil, nil
	}
	h := f.Macs[i]
	var own []finding.Finding
	for _, fd := range machines.Findings(f) {
		if fd.Machine.On == h.Name {
			own = append(own, fd)
		}
	}
	vms := []overview.VM{}
	for _, vm := range machines.Ordered(h.VMs) {
		vms = append(vms, overview.VM{Name: vm.Name, Running: vm.State == machines.Running})
	}
	return own, vms
}
