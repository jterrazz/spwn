package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
)

// value: the agents of every running workspace of every Mac that answers,
// read as in asks, each with its VM and its Mac; a workspace that does not
// answer adds none. And the names no agent read holds: every workspace's
// of every Mac, a silent one's too, and the agents a workspace held when it
// last stopped, of one stopped or of a silent Mac.
func everyMachine(ctx context.Context, r call.Router, in ListIn) ([]agents.Agent, agents.Held, error) {
	read, err := call.Read(ctx, r, machine.Ref{}, machinescalls.List, machinescalls.ListIn{Unwritten: true}, nil)
	if err != nil {
		return nil, agents.Held{}, err
	}
	var out []agents.Agent
	var held agents.Held
	for _, mac := range read.All.Macs {
		for _, vm := range mac.VMs {
			if vm.Lifetime.Kind() != machine.Workspace {
				continue
			}
			held.Workspaces = append(held.Workspaces, vm.Name)
			if !mac.Reachable || vm.State != machines.Running {
				held.Agents = append(held.Agents, vm.Agents...)
				continue
			}
			var kept []agents.Agent
			if err := r.Invoke(ctx, machine.Ref{Name: vm.Name}, "agents.list", ListIn{AsGiven: in.AsGiven}, &kept, nil); err != nil {
				continue
			}
			for _, a := range kept {
				a.VM, a.Mac = vm.Name, mac.Name
				out = append(out, a)
			}
		}
	}
	return out, held, nil
}

func HeldEverywhere(ctx context.Context, r call.Router) (agents.Held, error) {
	// value: the names every machine holds, read from a Mac across every
	// machine: what its engine answers a workspace that asks.
	all, h, err := everyMachine(ctx, r, ListIn{AsGiven: true})
	if err != nil {
		return agents.Held{}, err
	}
	for _, a := range all {
		if a.Name != "" {
			h.Agents = append(h.Agents, a.Name)
		}
	}
	return h, nil
}

// value: the names a start on this machine is judged against: every
// machine's, read across them from a Mac, asked of its Mac inside a
// workspace.
func (d Deps) held(ctx context.Context) (agents.Held, error) {
	if d.Kind == machine.Computer {
		return HeldEverywhere(ctx, d.Calls)
	}
	if d.Held == nil {
		return agents.Held{}, nil
	}
	return d.Held(ctx)
}

// value: a call of a Mac that names the Mac itself as the workspace it
// reaches, refused as the workspace-only gate refuses it.
func (d Deps) notAWorkspace(name string) error {
	v := agents.Applies(machine.Profile{Ref: machine.Ref{Name: d.Machine}, Kind: d.Kind})
	return fail.New(fail.NotForThisMachine, call.NotForThisMachine, d.Machine, map[string]string{"call": name, "why": v.Why, "fix": v.Fix})
}
