package wire

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/agents"
	agentscalls "github.com/jterrazz/studio-station/src/agents/calls"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	enginecalls "github.com/jterrazz/studio-station/src/engine/calls"
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
	// value: the names every machine holds as this machine's engine reads them,
	// which a workspace asks its Mac's.
	Engine func(ctx context.Context) (agents.Held, error)
}

func agentsCommand(env kit.Env, where page.Context, d agentsDeps) *cobra.Command {
	f := agentsEverywhere{env: env, engine: d.Engine}
	return agentscmd.Command(agentscmd.Deps{ThisMachine: where, Everywhere: f.all}, env,
		agentsrecord.Command(agentsrecord.Deps{Terminal: d.Terminal}, env),
		agentsattach.Command(agentsattach.Deps{Attach: d.Attach, Find: f.find}, env),
		agentsreply.Command(agentsreply.Deps{ThisMachine: where, Where: f.where}, env),
		agentsstart.Command(agentsstart.Deps{ThisMachine: where, Held: f.held}, env),
		agentsmerge.Command(agentsmerge.Deps{ThisMachine: where, Where: f.where}, env),
		agentsstop.Command(agentsstop.Deps{ThisMachine: where, Where: f.where}, env),
		agentsforget.Command(agentsforget.Deps{ThisMachine: where, Where: f.where}, env))
}

// invariant: SPEC B5.22: an agent is reached by its name alone: on a Mac,
// every machine is read for it; with --on, or inside a workspace, it is
// that workspace's.
type agentsEverywhere struct {
	env    kit.Env
	engine func(ctx context.Context) (agents.Held, error)
}

func (f agentsEverywhere) all(ctx context.Context) ([]agents.Agent, error) {
	// value: the agents of every running workspace of every Mac that
	// answers, each with its Mac; a workspace that does not answer adds none.
	out, _, err := f.everyMachine(ctx)
	return out, err
}

// value: the agents of every running workspace of every Mac that answers,
// and the name of every workspace of those Macs, running or not.
func (f agentsEverywhere) everyMachine(ctx context.Context) ([]agents.Agent, []string, error) {
	read, err := call.Read(ctx, f.env.Calls, f.env.Self, machinescalls.List, machinescalls.ListIn{Unwritten: true}, nil)
	if err != nil {
		return nil, nil, err
	}
	var out []agents.Agent
	var workspaces []string
	for _, mac := range read.All.Macs {
		if !mac.Reachable {
			continue
		}
		for _, vm := range mac.VMs {
			if vm.Lifetime.Kind() != coremachine.Workspace {
				continue
			}
			workspaces = append(workspaces, vm.Name)
			if vm.State != model.Running {
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
	return out, workspaces, nil
}

// invariant: SPEC B5.22: a new agent's name is checked against every
// machine's whatever path starts it: a Mac reads every Mac, a workspace
// asks its Mac's engine, which reads it from there. A workspace whose Mac's
// engine is down checks its own agents alone: no Mac reaches its agents
// meanwhile, and a find refuses a name two agents hold once one does.
func (f agentsEverywhere) held(ctx context.Context) (agents.Held, error) {
	if f.env.Kind != coremachine.Mac {
		if f.engine == nil {
			return agents.Held{}, nil
		}
		h, err := f.engine(ctx)
		if down := new(enginecalls.DownError); errors.As(err, &down) {
			return agents.Held{}, nil
		}
		return h, err
	}
	return f.heldEverywhere(ctx)
}

func (f agentsEverywhere) heldEverywhere(ctx context.Context) (agents.Held, error) {
	all, workspaces, err := f.everyMachine(ctx)
	if err != nil {
		return agents.Held{}, err
	}
	h := agents.Held{Workspaces: workspaces}
	for _, a := range all {
		h.Agents = append(h.Agents, a.Name)
	}
	return h, nil
}

func (f agentsEverywhere) onThisMac() bool { return !f.env.Far() && f.env.Kind == coremachine.Mac }

// invariant: SPEC B5.22: on a Mac an agent is found among every machine's by
// its name; a name two agents hold is refused, both named with their
// workspace, so a gesture never reaches the other one.
func (f agentsEverywhere) find(ctx context.Context, name string) (agents.Agent, coremachine.Ref, error) {
	if !f.onThisMac() {
		kept, err := call.Invoke(ctx, f.env.Calls, f.env.Target(), agentscalls.List, struct{}{}, nil)
		if err != nil {
			return agents.Agent{}, coremachine.Ref{}, err
		}
		i := slices.IndexFunc(kept, func(a agents.Agent) bool { return a.Name == name })
		if i < 0 {
			return agents.Agent{}, coremachine.Ref{}, fail.New(fail.NotFound, agentscalls.UnknownAgent, name, nil)
		}
		return kept[i], f.env.Target(), nil
	}
	all, err := f.all(ctx)
	if err != nil {
		return agents.Agent{}, coremachine.Ref{}, err
	}
	var found []agents.Agent
	for _, a := range all {
		if a.Name == name {
			found = append(found, a)
		}
	}
	switch len(found) {
	case 0:
		return agents.Agent{}, coremachine.Ref{}, fail.New(fail.NotFound, agentscalls.UnknownAgent, name, nil)
	case 1:
		return found[0], coremachine.Ref{Name: found[0].VM}, nil
	}
	on := make([]string, len(found))
	for i, a := range found {
		on[i] = a.VM
	}
	return agents.Agent{}, coremachine.Ref{}, fail.New(fail.Refused, agentscalls.Ambiguous, name, map[string]string{"on": strings.Join(on, ", ")})
}

func (f agentsEverywhere) where(ctx context.Context, name string) (coremachine.Ref, error) {
	// value: with --on, or inside a workspace, the target itself, which says
	// whether it holds the agent.
	if !f.onThisMac() {
		return f.env.Target(), nil
	}
	_, at, err := f.find(ctx, name)
	return at, err
}
