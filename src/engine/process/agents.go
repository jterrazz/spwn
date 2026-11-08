package process

import (
	"cmp"
	"context"
	"encoding/json"
	"slices"

	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/engine"
)

// invariant: SPEC B5.1: the engine keeps the last report of each VM's
// agents in its memory alone, and tells once of an agent that turns to
// wait on the owner, or that finishes; the first report it hears of a VM
// is what its agents were, told of nothing.
func (d *Engine) heardAgents(ctx context.Context, v *vmState, now []agents.Agent) {
	d.mu.Lock()
	was, first := v.agents, !v.reported
	v.agents, v.reported = now, true
	d.mu.Unlock()
	if first {
		return
	}
	for _, a := range now {
		i := slices.IndexFunc(was, func(b agents.Agent) bool { return b.Session == a.Session || (a.Name != "" && b.Name == a.Name) })
		var before agents.Agent
		if i >= 0 {
			before = was[i]
		}
		switch {
		case a.State == agents.Waiting && (before.State != agents.Waiting || !sameAsk(before.Ask, a.Ask)):
			d.tell(ctx, v.rec.Name, waiting(v.rec.Name, a))
		case a.State == agents.Finished && i >= 0 && before.State != agents.Finished:
			d.tell(ctx, v.rec.Name, finished(v.rec.Name, a))
		}
	}
}

func (d *Engine) AgentsHeld(ctx context.Context) (agents.Held, error) {
	// invariant: SPEC B5.22: a workspace names an agent against the names
	// every machine holds, which only its Mac can read.
	if d.o.AgentsHeld == nil {
		return agents.Held{}, nil
	}
	return d.o.AgentsHeld(ctx)
}

func sameAsk(a, b *agents.Ask) bool {
	return (a == nil) == (b == nil) && (a == nil || (a.Title == b.Title && a.At.Equal(b.At)))
}

// value: SPEC B5.22: an agent's notice is keyed by its name; one no start
// named is its VM's.
func waiting(vm string, a agents.Agent) engine.Told {
	t := engine.Told{Kind: engine.NoticeAgentWaiting, Parts: []string{cmp.Or(a.Name, vm) + " is waiting"}}
	if a.Ask != nil && a.Ask.Title != "" {
		t.Parts = append(t.Parts, a.Ask.Title)
	}
	return t
}

func finished(vm string, a agents.Agent) engine.Told {
	t := engine.Told{Kind: engine.NoticeAgentFinished, Parts: []string{cmp.Or(a.Name, vm) + " is finished"}}
	if a.Reason != "" {
		t.Parts = append(t.Parts, a.Reason)
	}
	return t
}

// value: a report as memory keeps it, and back.
func keptReport(r []agents.Agent) json.RawMessage {
	if r == nil {
		return nil
	}
	data, _ := json.Marshal(r)
	return data
}

func readReport(raw json.RawMessage) []agents.Agent {
	var r []agents.Agent
	_ = json.Unmarshal(raw, &r)
	return r
}
