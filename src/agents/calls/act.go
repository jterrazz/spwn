package calls

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/confirm"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/machine"
)

const (
	UnknownAgent fail.Code = "agents.unknown-agent"
	UnknownApp   fail.Code = "agents.unknown-app"
	NoPR         fail.Code = "agents.no-pr"
	Ended        fail.Code = "agents.ended"
	NoText       fail.Code = "agents.no-text"
	NoTerminal   fail.Code = "agents.no-terminal"
	NameTaken    fail.Code = "agents.name-taken"
	BadName      fail.Code = "agents.bad-name"
	StillLive    fail.Code = "agents.live"
	// value: a start that names neither a workspace nor a new one, or both.
	NoWhere fail.Code = "agents.no-where"
	// value: an agent its workspace's stop or delete could not stop first;
	// the VM goes on.
	NotStopped fail.Code = "agents.not-stopped"
)

var List = call.Call[Deps, struct{}, []agents.Agent]{
	Name: "agents.list", Where: call.AtTarget, Applies: agents.Applies,
	Run: func(ctx context.Context, d Deps, _ struct{}, _ call.Progress) ([]agents.Agent, error) {
		// invariant: the VM's agents as its journal holds them, by session,
		// then those its multiplexer alone sees, each named.
		return d.agents(ctx)
	},
}

// value: the journal's agents, then those of the multiplexer's live terminals no
// hook reports, each with its name; a multiplexer that cannot be read adds none.
func (d Deps) agents(ctx context.Context) ([]agents.Agent, error) {
	kept, err := d.Journal.Read(ctx)
	if err != nil {
		return nil, err
	}
	if d.Session != nil {
		if live, err := d.Session.Terminals(ctx); err == nil {
			kept = agents.Detected(kept, live)
		}
	}
	return agents.Named(kept, d.Machine), nil
}

// value: the agent of that name, refused when the VM holds none.
func (d Deps) find(ctx context.Context, name string) (agents.Agent, error) {
	kept, err := d.agents(ctx)
	if err != nil {
		return agents.Agent{}, err
	}
	i := slices.IndexFunc(kept, func(a agents.Agent) bool { return a.Name == name })
	if i < 0 {
		return agents.Agent{}, fail.New(fail.NotFound, UnknownAgent, name, nil)
	}
	return kept[i], nil
}

// value: the agent of that name, refused when it finished, for a gesture
// that needs it running.
func (d Deps) running(ctx context.Context, name string) (agents.Agent, error) {
	a, err := d.find(ctx, name)
	if err != nil {
		return a, err
	}
	if !agents.Live(a) {
		return a, fail.New(fail.Refused, Ended, name, map[string]string{"reason": a.Reason})
	}
	return a, nil
}

type ReplyIn struct {
	Agent string
	// value: what the owner types to the agent; it travels in this input and
	// on the terminal's stdin, never on a command line.
	Text string
}

var Reply = call.Call[Deps, ReplyIn, agents.Agent]{
	Name: "agents.reply", Where: call.AtTarget, Applies: agents.Applies,
	Errors: []fail.Code{NoText, UnknownAgent, Ended, NoTerminal},
	Run: func(ctx context.Context, d Deps, in ReplyIn, _ call.Progress) (agents.Agent, error) {
		// invariant: a reply is typed into the agent's terminal, then the agent
		// works on it: its ask is answered before its hook says so.
		if strings.TrimSpace(in.Text) == "" {
			return agents.Agent{}, fail.New(fail.Refused, NoText, in.Agent, nil)
		}
		a, err := d.running(ctx, in.Agent)
		if err != nil {
			return a, err
		}
		if a.Terminal == "" {
			return a, fail.New(fail.Refused, NoTerminal, a.Name, nil)
		}
		if err := d.Session.Send(ctx, a.Terminal, in.Text); err != nil {
			return a, err
		}
		a = agents.Judge(a, agents.Event{Kind: agents.Prompted, Session: a.Session, At: d.Now()})
		return a, d.Journal.Write(ctx, a)
	},
}

type StartIn struct {
	// value: the app whose agent starts; empty is the first station reads.
	App agents.App
	// value: its name; empty is the next its VM gives.
	Name string
	// value: the folder it works in, inside the VM; empty is the terminal's
	// own.
	Folder string
	// value: its first prompt, typed once it starts, never on its command
	// line; empty, it waits for one.
	Task string
}

var Start = call.Call[Deps, StartIn, agents.Agent]{
	Name: "agents.start", Where: call.AtTarget, Applies: agents.Applies,
	Errors: []fail.Code{UnknownApp, BadName, NameTaken},
	Run: func(ctx context.Context, d Deps, in StartIn, _ call.Progress) (agents.Agent, error) {
		// invariant: SPEC B5.22: one agent is one terminal of its own on the
		// VM's multiplexer, named; the journal keeps it from its first frame,
		// with its task, through the fold its hook's events take, and its task
		// is typed as its first prompt once it is ready.
		app := in.App
		if app == "" && len(d.Hooks) > 0 {
			app = d.Hooks[0].App()
		}
		hooks := d.hooksOf(app)
		if hooks == nil {
			return agents.Agent{}, fail.New(fail.NotFound, UnknownApp, string(in.App), nil)
		}
		kept, err := d.agents(ctx)
		if err != nil {
			return agents.Agent{}, err
		}
		taken := make([]string, 0, len(kept))
		for _, a := range kept {
			taken = append(taken, a.Name)
		}
		name := in.Name
		switch {
		case name == "":
			name = agents.NameFor(d.Machine, taken)
		case !agents.ValidName(name):
			return agents.Agent{}, fail.New(fail.Refused, BadName, name, nil)
		case agents.Clashes(taken, name):
			return agents.Agent{}, fail.New(fail.Refused, NameTaken, name, nil)
		}
		terminal, err := d.Session.Open(ctx, app, name, in.Folder, hooks.Launch())
		if err != nil {
			return agents.Agent{}, err
		}
		a := agents.Judge(agents.Agent{VM: d.Machine}, agents.Event{Kind: agents.Began, App: app, Name: name, Session: agents.SessionOf(name), Terminal: terminal, Folder: in.Folder, Prompt: in.Task, At: d.Now()})
		if err := d.Journal.Write(ctx, a); err != nil {
			return a, err
		}
		if strings.TrimSpace(in.Task) == "" {
			return a, nil
		}
		if err := d.Session.Send(ctx, terminal, in.Task); err != nil {
			return a, err
		}
		a = agents.Judge(a, agents.Event{Kind: agents.Prompted, Session: a.Session, At: d.Now()})
		return a, d.Journal.Write(ctx, a)
	},
}

type AgentIn struct {
	Agent string
}

var Merge = call.Call[Deps, AgentIn, agents.Agent]{
	// invariant: owner decision D2 (SPEC §D3), built on its recommendation:
	// the merge asks no confirmation, the gesture names the PR.
	Name: "agents.merge", Where: call.AtTarget, Applies: agents.Applies,
	Errors: []fail.Code{UnknownAgent, Ended, NoPR},
	Run: func(ctx context.Context, d Deps, in AgentIn, _ call.Progress) (agents.Agent, error) {
		// invariant: the agent's PR is merged in its folder with the VM's own
		// forge, and the agent is finished with it.
		a, err := d.running(ctx, in.Agent)
		if err != nil {
			return a, err
		}
		if a.PR == nil {
			return a, fail.New(fail.Refused, NoPR, a.Name, nil)
		}
		if err := d.Forge.Merge(ctx, a.Folder, a.PR.Number); err != nil {
			return a, err
		}
		a = agents.Judge(a, agents.Event{Kind: agents.Ended, Session: a.Session, At: d.Now(), Reason: "merged PR #" + strconv.Itoa(a.PR.Number)})
		return a, d.Journal.Write(ctx, a)
	},
}

type StopIn struct {
	Agent string
	// value: why it stops, as its finish says it; empty is "stopped".
	Reason  string
	Confirm string
}

var Stop = call.Call[Deps, StopIn, agents.Agent]{
	Name: "agents.stop", Where: call.AtTarget, Applies: agents.Applies, Confirm: confirm.StopAgent,
	Errors: []fail.Code{UnknownAgent, Ended, confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in StopIn, _ call.Progress) (agents.Agent, error) {
		// invariant: SPEC B5.22: stopping an agent closes its terminal and
		// finishes it; its VM is never touched.
		a, err := d.running(ctx, in.Agent)
		if err != nil {
			return a, err
		}
		if err := confirm.Ask(confirm.StopAgent, machine.Workspace, a.Name, in.Confirm); err != nil {
			return a, err
		}
		if a.Terminal != "" {
			if err := d.Session.Close(ctx, a.Terminal); err != nil {
				return a, err
			}
		}
		a = agents.Judge(a, agents.Event{Kind: agents.Ended, Session: a.Session, At: d.Now(), Reason: cmp.Or(in.Reason, "stopped")})
		return a, d.Journal.Write(ctx, a)
	},
}

var Forget = call.Call[Deps, AgentIn, agents.Agent]{
	Name: "agents.forget", Where: call.AtTarget, Applies: agents.Applies,
	Errors: []fail.Code{UnknownAgent, StillLive},
	Run: func(ctx context.Context, d Deps, in AgentIn, _ call.Progress) (agents.Agent, error) {
		// invariant: SPEC B5.9: only a finished agent is forgotten: its
		// journal entry goes, and its terminal if Herdr still holds it; its
		// VM stays.
		a, err := d.find(ctx, in.Agent)
		if err != nil {
			return a, err
		}
		if agents.Live(a) {
			return a, fail.New(fail.Refused, StillLive, a.Name, map[string]string{"state": string(a.State)})
		}
		if a.Terminal != "" && d.Session != nil {
			if live, err := d.Session.Terminals(ctx); err == nil && slices.ContainsFunc(live, func(t agents.Terminal) bool { return t.Address == a.Terminal }) {
				if err := d.Session.Close(ctx, a.Terminal); err != nil {
					return a, err
				}
			}
		}
		return a, d.Journal.Forget(ctx, a.Session)
	},
}
