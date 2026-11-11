package wire

import (
	"context"
	"errors"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/agents"
	agentscalls "github.com/jterrazz/studio-station/src/agents/calls"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	enginecalls "github.com/jterrazz/studio-station/src/engine/calls"
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
	return agentscmd.Command(agentscmd.Deps{ThisMachine: where}, env,
		agentsrecord.Command(agentsrecord.Deps{Terminal: d.Terminal}, env),
		agentsattach.Command(agentsattach.Deps{Attach: d.Attach, Find: f.find}, env),
		agentsreply.Command(agentsreply.Deps{ThisMachine: where, Where: f.where}, env),
		agentsstart.Command(agentsstart.Deps{ThisMachine: where, At: func() coremachine.Ref { return agentscmd.At(env) }}, env),
		agentsmerge.Command(agentsmerge.Deps{ThisMachine: where, Where: f.where}, env),
		agentsstop.Command(agentsstop.Deps{ThisMachine: where, Where: f.where}, env),
		agentsforget.Command(agentsforget.Deps{ThisMachine: where, Where: f.where}, env))
}

// invariant: SPEC B5.22: an agent is reached by its name alone: on a Mac,
// among every machine's, as agents.list reads them; with --on, or inside a workspace, it
// is that workspace's.
type agentsEverywhere struct {
	env kit.Env
}

func (f agentsEverywhere) onThisMac() bool { return !f.env.Far() && f.env.Kind == coremachine.Computer }

// invariant: SPEC B5.22: on a Mac an agent is found among every machine's by
// its name, with --on or inside a workspace among that workspace's; a name
// two agents hold is refused, both named with their workspace, so a gesture
// never reaches the other one.
func (f agentsEverywhere) find(ctx context.Context, name string) (agents.Agent, coremachine.Ref, error) {
	all, err := call.Invoke(ctx, f.env.Calls, agentscmd.At(f.env), agentscalls.List, agentscmd.Scope(f.env), nil)
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
		if !f.onThisMac() {
			return found[0], f.env.Target(), nil
		}
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

// value: the names every machine holds, which a workspace asks its Mac's
// engine within heldBudget; a Mac whose engine is down holds none, so the
// workspace checks its own agents alone: no Mac reaches them meanwhile, and
// a find refuses a name two agents hold once one does. Nil on a Mac, which
// reads every machine itself.
func (m machine) heldOnEveryMachine(kind coremachine.Kind) func(ctx context.Context) (agents.Held, error) {
	if kind != coremachine.Workspace {
		return nil
	}
	return func(ctx context.Context) (agents.Held, error) {
		ctx, cancel := context.WithTimeout(ctx, heldBudget)
		defer cancel()
		h, err := m.engineAPI().AgentsHeld(ctx)
		if down := new(enginecalls.DownError); errors.As(err, &down) {
			return agents.Held{}, nil
		}
		return h, err
	}
}

// value: the Router a call reaches another machine through: this run's,
// once it is built.
type farRouter struct{ m machine }

func (f farRouter) Invoke(ctx context.Context, target coremachine.Ref, name string, in, out any, p call.Progress) error {
	if f.m.far == nil || f.m.far.router == nil {
		return orNone(nil)
	}
	return f.m.far.router.Invoke(ctx, target, name, in, out, p)
}

func (f farRouter) Watch(ctx context.Context, target coremachine.Ref, name string, in any, p call.Progress) (<-chan call.Update[any], error) {
	if f.m.far == nil || f.m.far.router == nil {
		return nil, orNone(nil)
	}
	return f.m.far.router.Watch(ctx, target, name, in, p)
}
