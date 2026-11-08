package feeds

import (
	"context"

	"github.com/jterrazz/studio-station/src/agents"
	agentscalls "github.com/jterrazz/studio-station/src/agents/calls"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/engine"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	"github.com/jterrazz/studio-station/src/page"
	pagemenubar "github.com/jterrazz/studio-station/src/pages/menubar"
)

func (o Menubar) Agent() engine.Feed {
	// invariant: SPEC B3: menubar.agent:<name> is one agent, read with the
	// fleet's agents through this Mac's engine, which asks each machine:
	// when it opens, every 30 s, on a refresh and on a change of this
	// engine's state.
	return func(ctx context.Context, in engine.FeedInput) <-chan any {
		read := func(ctx context.Context) page.Page {
			r := pagemenubar.Agent{Name: in.Subject, Now: o.Clock.Now(), Asks: levels}
			listed, err := call.Read(ctx, o.Calls, machine.Ref{}, machinescalls.List, machinescalls.ListIn{Unwritten: true}, nil)
			if err != nil {
				return pagemenubar.Unread(pagemenubar.AgentID, o.ThisMachine, err)
			}
			kept, err := call.Invoke(ctx, o.Calls, machine.Ref{}, agentscalls.List, agentscalls.ListIn{}, nil)
			if err != nil {
				return pagemenubar.Unread(pagemenubar.AgentID, o.ThisMachine, err)
			}
			r.All, r.Agents = &listed.All, kept
			if r.Agents == nil {
				r.Agents = []agents.Agent{}
			}
			return pagemenubar.AgentPage(r)
		}
		return o.reread(ctx, in, pagemenubar.AgentID, remoteEvery, read)
	}
}
