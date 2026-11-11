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
	"github.com/jterrazz/studio-station/src/machines"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
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
	// value: a start that names neither a workspace nor a new one, or both,
	// or a new one from inside a workspace.
	NoWhere fail.Code = "agents.no-where"
	// value: a name two agents of two machines hold, each started inside its
	// own workspace.
	Ambiguous fail.Code = "agents.ambiguous"
	// value: an agent its workspace's stop or delete could not stop first;
	// the VM goes on.
	NotStopped fail.Code = "agents.not-stopped"
)

type ListIn struct {
	// value: each agent with the name its start gave it, one run by hand
	// with none: the read every machine's names are made of, which never
	// asks every machine in turn.
	AsGiven bool `json:",omitempty"`
	// value: the workspace whose agents a Mac reads; empty, every machine's.
	// Inside a workspace it is the workspace itself, whatever name its Mac
	// knows it by.
	On string `json:",omitempty"`
}

var List = call.Call[Deps, ListIn, []agents.Agent]{
	Name: "agents.list", Where: call.AtTarget, Applies: agents.Reaches,
	Run: func(ctx context.Context, d Deps, in ListIn, _ call.Progress) ([]agents.Agent, error) {
		// invariant: SPEC B5.22: on a Mac, every agent of every machine, each
		// with its VM and its Mac; with On, the agents of that workspace, read
		// inside it: as its journal holds them, by session, then those its
		// multiplexer alone sees, each named.
		if d.Kind == machine.Computer && in.On == "" {
			out, _, err := everyMachine(ctx, d.Calls, in)
			return out, err
		}
		if d.Kind == machine.Computer {
			if in.On == d.Machine {
				return nil, d.notAWorkspace("agents.list")
			}
			var out []agents.Agent
			err := d.Calls.Invoke(ctx, machine.Ref{Name: in.On}, "agents.list", in, &out, nil)
			return out, err
		}
		if in.AsGiven {
			return d.given(ctx)
		}
		return d.agents(ctx)
	},
}

// value: the journal's agents, then those of the multiplexer's live terminals no
// hook reports, each with its VM and the name its start gave, none for one
// run by hand; a multiplexer that cannot be read adds none.
func (d Deps) given(ctx context.Context) ([]agents.Agent, error) {
	kept, err := d.Journal.Read(ctx)
	if err != nil {
		return nil, err
	}
	if d.Session != nil {
		if live, err := d.Session.Terminals(ctx); err == nil {
			kept = agents.Detected(kept, live)
		}
	}
	for i := range kept {
		kept[i].VM = cmp.Or(kept[i].VM, d.Machine)
	}
	return kept, nil
}

// invariant: SPEC B5.22: an agent run by hand takes a name no machine
// holds, as the workspace's Mac reads them; a Mac that does not answer
// leaves the workspace's own names alone.
func (d Deps) agents(ctx context.Context) ([]agents.Agent, error) {
	kept, err := d.given(ctx)
	if err != nil {
		return nil, err
	}
	var elsewhere []string
	if d.Held != nil && slices.ContainsFunc(kept, func(a agents.Agent) bool { return a.Name == "" }) {
		if held, err := d.Held(ctx); err == nil {
			elsewhere = held.Agents
		}
	}
	return agents.Named(kept, d.Machine, elsewhere), nil
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
	// value: its first prompt, typed once it starts, never on its command
	// line; empty, it waits for one.
	Task string
	// value: its name; empty is the next its workspace gives.
	Name string
	// value: the app whose agent starts; empty is the first station reads.
	App agents.App
	// value: the running workspace a Mac starts it on; inside a workspace,
	// the workspace itself, whatever name its Mac knows it by.
	On string `json:",omitempty"`
	// value: a new throwaway it starts on, made first; a Mac asks exactly one
	// of On and New.
	New *New `json:",omitempty"`
	// value: the folder it works in, inside the VM; empty is the terminal's
	// own.
	Folder string `json:",omitempty"`
}

type New struct {
	// value: the Mac the throwaway is made on; empty is the one asked.
	Mac   string `json:",omitempty"`
	Image string `json:",omitempty"`
	CPUs  int    `json:",omitempty"`
	// value: memory and disk with their unit, e.g. 8G; empty keeps the
	// image's default.
	Memory string `json:",omitempty"`
	Disk   string `json:",omitempty"`
	// value: the answer to "build the missing image first?".
	Confirm string `json:",omitempty"`
}

var Start = call.Call[Deps, StartIn, agents.Agent]{
	Name: "agents.start", Where: call.AtTarget, Applies: agents.Reaches,
	Errors: []fail.Code{UnknownApp, BadName, NameTaken, NoWhere},
	Run: func(ctx context.Context, d Deps, in StartIn, p call.Progress) (agents.Agent, error) {
		// invariant: SPEC B5.22: a Mac starts an agent on the running workspace
		// On names, or on the throwaway New makes first, its name unique on
		// every machine; inside a workspace it starts on the workspace itself.
		if d.Kind == machine.Computer {
			return d.startFromMac(ctx, in, p)
		}
		switch {
		case in.New != nil && in.On != "":
			return agents.Agent{}, fail.New(fail.Refused, NoWhere, "agents.start", map[string]string{"both": "both"})
		case in.New != nil:
			return agents.Agent{}, fail.New(fail.Refused, NoWhere, "agents.start", map[string]string{"both": "inside"})
		}
		return d.startHere(ctx, in)
	},
}

// invariant: SPEC B5.22: a name is judged against every machine's before
// anything is made: an agent's anywhere, or another workspace's own; the
// throwaway New asks is made only then, and its agent started on it once
// it answers.
func (d Deps) startFromMac(ctx context.Context, in StartIn, p call.Progress) (agents.Agent, error) {
	switch {
	case (in.On == "") == (in.New == nil):
		both := "neither"
		if in.New != nil {
			both = "both"
		}
		return agents.Agent{}, fail.New(fail.Refused, NoWhere, "agents.start", map[string]string{"both": both})
	case in.On == d.Machine:
		return agents.Agent{}, d.notAWorkspace("agents.start")
	case in.Name != "" && !agents.ValidName(in.Name):
		return agents.Agent{}, fail.New(fail.Refused, BadName, in.Name, nil)
	}
	held, err := d.held(ctx)
	if err != nil {
		return agents.Agent{}, err
	}
	vm := in.On
	if in.New != nil {
		vm = in.Name
	}
	if ws, taken := held.Takes(in.Name, vm); in.Name != "" && taken {
		return agents.Agent{}, nameTaken(in.Name, ws)
	}
	if in.New != nil {
		if vm, err = d.create(ctx, in, p); err != nil {
			return agents.Agent{}, err
		}
	}
	at := in
	at.On, at.New = vm, nil
	if at.Name == "" {
		at.Name = agents.NameFor(vm, held.Agents)
	}
	var a agents.Agent
	err = d.Calls.Invoke(ctx, machine.Ref{Name: vm}, "agents.start", at, &a, nil)
	if err != nil && in.New != nil {
		return a, &machines.AgentNotStartedError{Name: vm, Cause: err}
	}
	return a, err
}

// value: the throwaway made for the agent, by its name, on the Mac New
// asks; its steps said as they run.
func (d Deps) create(ctx context.Context, in StartIn, p call.Progress) (string, error) {
	size, err := machines.ParseResources(in.New.CPUs, in.New.Memory, in.New.Disk)
	if err != nil {
		return "", err
	}
	created, err := call.Invoke(ctx, d.Calls, machine.Ref{Name: in.New.Mac}, machinescalls.CreateWorkspace, machinescalls.CreateIn{
		Request: machines.CreateRequest{Image: cmp.Or(in.New.Image, "linux"), Name: in.Name, Lifetime: machines.Throwaway, Size: size},
		Confirm: in.New.Confirm,
	}, p)
	if err != nil {
		return "", err
	}
	return created.VM.Name, nil
}

// invariant: SPEC B5.22: one agent is one terminal of its own on the VM's
// multiplexer, named, its name taken by no agent of any machine nor another
// workspace's own; the journal keeps it from its first frame, with its task,
// through the fold its hook's events take, and its task is typed as its
// first prompt once it is ready.
func (d Deps) startHere(ctx context.Context, in StartIn) (agents.Agent, error) {
	app := in.App
	if app == "" && len(d.Hooks) > 0 {
		app = d.Hooks[0].App()
	}
	hooks := d.hooksOf(app)
	if hooks == nil {
		return agents.Agent{}, fail.New(fail.NotFound, UnknownApp, string(in.App), nil)
	}
	name := in.Name
	if name != "" && !agents.ValidName(name) {
		return agents.Agent{}, fail.New(fail.Refused, BadName, name, nil)
	}
	kept, err := d.agents(ctx)
	if err != nil {
		return agents.Agent{}, err
	}
	held, err := d.held(ctx)
	if err != nil {
		return agents.Agent{}, err
	}
	for _, a := range kept {
		held.Agents = append(held.Agents, a.Name)
	}
	if name == "" {
		name = agents.NameFor(d.Machine, held.Agents)
	} else if ws, taken := held.Takes(name, d.Machine); taken {
		return agents.Agent{}, nameTaken(name, ws)
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
}

// value: the refusal of a name in use: an agent's, or the workspace's whose
// own it is.
func nameTaken(name, workspace string) error {
	if workspace == "" {
		return fail.New(fail.Refused, NameTaken, name, nil)
	}
	return fail.New(fail.Refused, NameTaken, name, map[string]string{"workspace": workspace, "nth": workspace + agents.Nth + "<n>"})
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
