package wire

import (
	"context"
	"slices"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/agents"
	agentscalls "github.com/jterrazz/studio-station/src/agents/calls"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	model "github.com/jterrazz/studio-station/src/machines"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	"github.com/jterrazz/studio-station/src/page"
	agentscmd "github.com/jterrazz/studio-station/src/surface/cli/agents"
	agentsattach "github.com/jterrazz/studio-station/src/surface/cli/agents/attach"
	agentsforget "github.com/jterrazz/studio-station/src/surface/cli/agents/forget"
	agentsmerge "github.com/jterrazz/studio-station/src/surface/cli/agents/merge"
	agentsrecord "github.com/jterrazz/studio-station/src/surface/cli/agents/record"
	agentsreply "github.com/jterrazz/studio-station/src/surface/cli/agents/reply"
	agentsstart "github.com/jterrazz/studio-station/src/surface/cli/agents/start"
	agentsstop "github.com/jterrazz/studio-station/src/surface/cli/agents/stop"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type agentsDeps struct {
	// value: the terminal of the multiplexer this process runs in, empty outside one.
	Terminal string
	Attach   func(ctx context.Context, machine, terminal string) error
}

func agentsCommand(env kit.Env, where page.Context, d agentsDeps) *cobra.Command {
	f := agentsEverywhere{env: env}
	return agentscmd.Command(agentscmd.Deps{ThisMachine: where, Everywhere: f.all}, env,
		agentsrecord.Command(agentsrecord.Deps{Terminal: d.Terminal}, env),
		agentsattach.Command(agentsattach.Deps{Attach: d.Attach, Find: f.find}, env),
		agentsreply.Command(agentsreply.Deps{ThisMachine: where, Where: f.where}, env),
		agentsstart.Command(agentsstart.Deps{ThisMachine: where, Everywhere: f.all}, env),
		agentsmerge.Command(agentsmerge.Deps{ThisMachine: where, Where: f.where}, env),
		agentsstop.Command(agentsstop.Deps{ThisMachine: where, Where: f.where}, env),
		agentsforget.Command(agentsforget.Deps{ThisMachine: where, Where: f.where}, env))
}

// invariant: SPEC B5.22: an agent is reached by its name alone: on a Mac,
// every machine is read for it; with --on, or inside a workspace, it is
// that workspace's.
type agentsEverywhere struct{ env kit.Env }

func (f agentsEverywhere) all(ctx context.Context) ([]agents.Agent, error) {
	// value: the agents of every running workspace of every Mac that
	// answers, each with its Mac; a workspace that does not answer adds none.
	read, err := call.Read(ctx, f.env.Calls, f.env.Self, machinescalls.List, machinescalls.ListIn{Unwritten: true}, nil)
	if err != nil {
		return nil, err
	}
	var out []agents.Agent
	for _, mac := range read.All.Macs {
		if !mac.Reachable {
			continue
		}
		for _, vm := range mac.VMs {
			if vm.Lifetime.Kind() != coremachine.Workspace || vm.State != model.Running {
				continue
			}
			kept, err := call.Invoke(ctx, f.env.Calls, coremachine.Ref{Name: vm.Name}, agentscalls.List, struct{}{}, nil)
			if err != nil {
				continue
			}
			for _, a := range kept {
				a.VM, a.Mac = vm.Name, mac.Name
				out = append(out, a)
			}
		}
	}
	return out, nil
}

func (f agentsEverywhere) find(ctx context.Context, name string) (agents.Agent, coremachine.Ref, error) {
	var kept []agents.Agent
	var err error
	if f.env.Far() || f.env.Kind != coremachine.Mac {
		kept, err = call.Invoke(ctx, f.env.Calls, f.env.Target(), agentscalls.List, struct{}{}, nil)
	} else {
		kept, err = f.all(ctx)
	}
	if err != nil {
		return agents.Agent{}, coremachine.Ref{}, err
	}
	i := slices.IndexFunc(kept, func(a agents.Agent) bool { return a.Name == name })
	if i < 0 {
		return agents.Agent{}, coremachine.Ref{}, fail.New(fail.NotFound, agentscalls.UnknownAgent, name, nil)
	}
	if f.env.Far() || f.env.Kind != coremachine.Mac {
		return kept[i], f.env.Target(), nil
	}
	return kept[i], coremachine.Ref{Name: kept[i].VM}, nil
}

func (f agentsEverywhere) where(ctx context.Context, name string) (coremachine.Ref, error) {
	// value: with --on, or inside a workspace, the target itself, which says
	// whether it holds the agent.
	if f.env.Far() || f.env.Kind != coremachine.Mac {
		return f.env.Target(), nil
	}
	_, at, err := f.find(ctx, name)
	return at, err
}
