package wire

import (
	"context"
	"slices"
	"strings"

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
	// answers, each with its Mac and its name across every machine; a workspace that
	// does not answer adds none.
	placed, err := f.placed(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]agents.Agent, len(placed))
	for i, p := range placed {
		out[i] = p.Agent
	}
	return out, nil
}

// value: an agent, by its name across every machine, and the name its
// own workspace knows it by.
type placed struct {
	agents.Agent
	local string
}

func (f agentsEverywhere) placed(ctx context.Context) ([]placed, error) {
	read, err := call.Read(ctx, f.env.Calls, f.env.Self, machinescalls.List, machinescalls.ListIn{Unwritten: true}, nil)
	if err != nil {
		return nil, err
	}
	var held []agents.Agent
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
				held = append(held, a)
			}
		}
	}
	named := agents.Unique(held)
	out := make([]placed, len(named))
	for i, a := range named {
		out[i] = placed{Agent: a, local: held[i].Name}
	}
	return out, nil
}

func (f agentsEverywhere) onThisMac() bool { return !f.env.Far() && f.env.Kind == coremachine.Mac }

func (f agentsEverywhere) find(ctx context.Context, name string) (agents.Agent, coremachine.Ref, error) {
	a, at, _, err := f.located(ctx, name)
	return a, at, err
}

// invariant: SPEC B5.22: on a Mac an agent is found among every machine's by
// its name there, and its workspace's calls are sent the name it knows it
// by; a name two agents hold is refused, both named with their workspace,
// so a gesture never reaches the other one.
func (f agentsEverywhere) located(ctx context.Context, name string) (agents.Agent, coremachine.Ref, string, error) {
	if !f.onThisMac() {
		kept, err := call.Invoke(ctx, f.env.Calls, f.env.Target(), agentscalls.List, struct{}{}, nil)
		if err != nil {
			return agents.Agent{}, coremachine.Ref{}, "", err
		}
		i := slices.IndexFunc(kept, func(a agents.Agent) bool { return a.Name == name })
		if i < 0 {
			return agents.Agent{}, coremachine.Ref{}, "", fail.New(fail.NotFound, agentscalls.UnknownAgent, name, nil)
		}
		return kept[i], f.env.Target(), name, nil
	}
	all, err := f.placed(ctx)
	if err != nil {
		return agents.Agent{}, coremachine.Ref{}, "", err
	}
	var found []placed
	for _, p := range all {
		if p.Name == name {
			found = append(found, p)
		}
	}
	switch len(found) {
	case 0:
		return agents.Agent{}, coremachine.Ref{}, "", fail.New(fail.NotFound, agentscalls.UnknownAgent, name, nil)
	case 1:
		return found[0].Agent, coremachine.Ref{Name: found[0].VM}, found[0].local, nil
	}
	on := make([]string, len(found))
	for i, p := range found {
		on[i] = p.VM
	}
	return agents.Agent{}, coremachine.Ref{}, "", fail.New(fail.Refused, agentscalls.Ambiguous, name, map[string]string{"on": strings.Join(on, ", ")})
}

func (f agentsEverywhere) where(ctx context.Context, name string) (coremachine.Ref, string, error) {
	// value: with --on, or inside a workspace, the target itself, which says
	// whether it holds the agent.
	if !f.onThisMac() {
		return f.env.Target(), name, nil
	}
	_, at, local, err := f.located(ctx, name)
	return at, local, err
}
