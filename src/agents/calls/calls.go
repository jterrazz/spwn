package calls

import (
	"context"

	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/agents/contract"
	"github.com/jterrazz/studio-station/src/core/call"
)

type Deps struct {
	// value: the hook of each app station reads, one strategy per app.
	Hooks   []contract.Hooks
	Journal contract.Journal
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.Bind(Record, d)}
}

type RecordIn struct {
	App agents.App
	// value: the event as the app's hook hands it over.
	Payload string
}

type Recorded struct {
	// value: false for an event no hook of the app reads: it is dropped.
	Kept    bool
	Session string       `json:",omitempty"`
	State   agents.State `json:",omitempty"`
}

var Record = call.Call[Deps, RecordIn, Recorded]{
	Name: "agents.record", Where: call.AtTarget, Applies: agents.Applies,
	Run: func(ctx context.Context, d Deps, in RecordIn, _ call.Progress) (Recorded, error) {
		// invariant: the journal's one writer: an event read by its app's
		// hook folds into its session, and anything else changes nothing.
		var hooks contract.Hooks
		for _, h := range d.Hooks {
			if h.App() == in.App {
				hooks = h
			}
		}
		if hooks == nil {
			return Recorded{}, nil
		}
		e, ok := hooks.Event(ctx, []byte(in.Payload))
		if !ok {
			return Recorded{}, nil
		}
		kept, err := d.Journal.Read(ctx)
		if err != nil {
			return Recorded{}, err
		}
		var a agents.Agent
		for _, k := range kept {
			if k.Session == e.Session {
				a = k
			}
		}
		a = agents.Judge(a, e)
		if err := d.Journal.Write(ctx, a); err != nil {
			return Recorded{}, err
		}
		return Recorded{Kept: true, Session: a.Session, State: a.State}, nil
	},
}
