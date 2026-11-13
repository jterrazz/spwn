package calls

import (
	"context"
	"slices"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

type Deps struct {
	// value: the hook of each app station reads, one strategy per app; the
	// first is the app an agent starts with when none is named.
	Hooks   []contract.Hooks
	Journal contract.Journal
	Session contract.Session
	Forge   contract.Forge
	Now     func() time.Time
	// value: the workspace these agents run in, by its name: the name its
	// first agent takes.
	Machine string
	// value: the names every machine holds, as this workspace's Mac reads
	// them; nil reads none.
	Held func(ctx context.Context) (agents.Held, error)
	// value: the kind of machine these calls run on: a Mac reads and starts
	// every machine's agents, a workspace its own.
	Kind machine.Kind
	// value: the Router a Mac reaches every machine through, its own
	// engine's calls included.
	Calls call.Router
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.Bind(Record, d), call.Bind(List, d), call.Bind(Reply, d), call.Bind(Start, d), call.Bind(Merge, d), call.Bind(Stop, d), call.Bind(Forget, d)}
}

func (d Deps) hooksOf(app agents.App) contract.Hooks {
	for _, h := range d.Hooks {
		if h.App() == app {
			return h
		}
	}
	return nil
}

type RecordIn struct {
	App agents.App
	// value: the terminal of the VM's multiplexer the hook ran in, empty outside one.
	Terminal string `json:",omitempty"`
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
		hooks := d.hooksOf(in.App)
		if hooks == nil {
			return Recorded{}, nil
		}
		e, ok := hooks.Event(ctx, []byte(in.Payload))
		if !ok {
			return Recorded{}, nil
		}
		if e.Terminal == "" {
			e.Terminal = in.Terminal
		}
		kept, err := d.Journal.Read(ctx)
		if err != nil {
			return Recorded{}, err
		}
		a := adopted(kept, e)
		was := a.Session
		a.Session = e.Session
		a = agents.Judge(a, e)
		if err := d.Journal.Write(ctx, a); err != nil {
			return Recorded{}, err
		}
		if was != "" && was != a.Session {
			if err := d.Journal.Forget(ctx, was); err != nil {
				return Recorded{}, err
			}
		}
		return Recorded{Kept: true, Session: a.Session, State: a.State}, nil
	},
}

// value: the agent an event folds into: the session its app named, else the
// live agent station started in the terminal the hook ran in, whose name
// and task the app's session takes.
func adopted(kept []agents.Agent, e agents.Event) agents.Agent {
	if i := slices.IndexFunc(kept, func(k agents.Agent) bool { return k.Session == e.Session }); i >= 0 {
		return kept[i]
	}
	if e.Terminal == "" {
		return agents.Agent{}
	}
	if i := slices.IndexFunc(kept, func(k agents.Agent) bool { return k.Terminal == e.Terminal && agents.Live(k) }); i >= 0 {
		return kept[i]
	}
	return agents.Agent{}
}
