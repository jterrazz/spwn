package kit

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/output"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

func (e Env) envelope(ctx context.Context) *output.Envelope {
	// invariant: the machine a document is about is the one its call ran on,
	// named as the interface names it (SPEC §2.5, §11.1); a name All machines
	// does not hold is said as it was typed.
	if e.Now == nil {
		return nil
	}
	out := &output.Envelope{At: e.Now()}
	target := e.Target()
	if target == e.Self {
		kind := e.Kind
		if kind == "" {
			kind = machine.Computer
		}
		out.Machine = output.Machine{Name: e.Self.Name, Label: e.Self.Name, Kind: string(kind), ThisMac: kind == machine.Computer}
		if kind == machine.Computer {
			out.Machine.Label = page.MacName(e.Self.Name, e.Self.Name)
		}
		return out
	}
	out.Machine = output.Machine{Name: target.Name, Label: target.Name, Kind: string(machine.Computer)}
	if e.Locate != nil {
		if ref, ok := e.Locate(target.Name); ok && ref.On == "" {
			out.Machine = output.Machine{Name: ref.Name, Label: page.MacName(ref.Name, e.Self.Name), Kind: string(machine.Computer)}
			return out
		}
	}
	if m, ok := e.named(ctx, target, machinescalls.ListIn{ThisMac: true}); ok {
		out.Machine = m
	} else if m, ok := e.named(ctx, target, machinescalls.ListIn{Unwritten: true}); ok {
		out.Machine = m
	}
	return out
}

func (e Env) named(ctx context.Context, target machine.Ref, in machinescalls.ListIn) (output.Machine, bool) {
	// invariant: a Mac and this Mac's own VMs are named from this Mac's
	// records; only a VM another Mac holds is asked of All machines.
	list, err := call.Read(ctx, e.Calls, e.Self, machinescalls.List, in, nil)
	if err != nil {
		return output.Machine{}, false
	}
	for _, h := range list.All.Macs {
		if h.Name == target.Name || page.MacName(h.Name, e.Self.Name) == target.Name {
			return output.Machine{Name: h.Name, Label: page.MacName(h.Name, e.Self.Name), Kind: string(machine.Computer), ThisMac: h.ThisMac}, true
		}
		for _, vm := range h.VMs {
			if vm.Name == target.Name {
				mac := h.Name
				return output.Machine{Name: vm.Name, Label: vm.Name, Kind: string(vm.Lifetime.Kind()), On: &mac}, true
			}
		}
	}
	return output.Machine{}, false
}
