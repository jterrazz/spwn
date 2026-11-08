package calls

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
)

const (
	UnknownSession fail.Code = "agents.unknown-session"
	UnknownApp     fail.Code = "agents.unknown-app"
	NoPR           fail.Code = "agents.no-pr"
	Ended          fail.Code = "agents.ended"
	NoText         fail.Code = "agents.no-text"
	NoTerminal     fail.Code = "agents.no-terminal"
)

var List = call.Call[Deps, struct{}, []agents.Agent]{
	Name: "agents.list", Where: call.AtTarget, Applies: agents.Applies,
	Run: func(ctx context.Context, d Deps, _ struct{}, _ call.Progress) ([]agents.Agent, error) {
		// invariant: the VM's agents as its journal holds them, by session,
		// then those its multiplexer alone sees.
		return d.agents(ctx)
	},
}

// value: the journal's agents, then those of the multiplexer's live terminals no
// hook reports; a multiplexer that cannot be read adds none.
func (d Deps) agents(ctx context.Context) ([]agents.Agent, error) {
	kept, err := d.Journal.Read(ctx)
	if err != nil || d.Session == nil {
		return kept, err
	}
	live, err := d.Session.Terminals(ctx)
	if err != nil {
		return kept, nil //nolint:nilerr // outside: a multiplexer that cannot be read leaves the journal's agents.
	}
	return agents.Detected(kept, live), nil
}

// value: the agent of a session the journal keeps, refused when it holds
// none, or when it ended, for a gesture that needs it running.
func (d Deps) running(ctx context.Context, session string) (agents.Agent, error) {
	kept, err := d.agents(ctx)
	if err != nil {
		return agents.Agent{}, err
	}
	i := slices.IndexFunc(kept, func(a agents.Agent) bool { return a.Session == session })
	if i < 0 {
		return agents.Agent{}, fail.New(fail.NotFound, UnknownSession, session, nil)
	}
	if kept[i].State == agents.Finished {
		return kept[i], fail.New(fail.Refused, Ended, session, map[string]string{"reason": kept[i].Reason})
	}
	return kept[i], nil
}

type ReplyIn struct {
	Session string
	// value: what the owner types to the agent; it travels in this input and
	// on the session's stdin, never on a command line.
	Text string
}

var Reply = call.Call[Deps, ReplyIn, agents.Agent]{
	Name: "agents.reply", Where: call.AtTarget, Applies: agents.Applies,
	Errors: []fail.Code{NoText, UnknownSession, Ended, NoTerminal},
	Run: func(ctx context.Context, d Deps, in ReplyIn, _ call.Progress) (agents.Agent, error) {
		// invariant: a reply is typed into the agent's session, then the agent
		// works on it: its ask is answered before its hook says so.
		if strings.TrimSpace(in.Text) == "" {
			return agents.Agent{}, fail.New(fail.Refused, NoText, in.Session, nil)
		}
		a, err := d.running(ctx, in.Session)
		if err != nil {
			return a, err
		}
		if a.Terminal == "" {
			return a, fail.New(fail.Refused, NoTerminal, a.Session, nil)
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
	// value: the folder it works in, inside the VM; empty is the session's
	// own.
	Folder string
	// value: its first prompt, typed once it starts, never on its command
	// line.
	Task string
}

type Started struct {
	App agents.App
	// value: the terminal of the VM's multiplexer it runs in.
	Terminal string `json:",omitempty"`
	Folder   string `json:",omitempty"`
	Task     string `json:",omitempty"`
}

var Start = call.Call[Deps, StartIn, Started]{
	Name: "agents.start", Where: call.AtTarget, Applies: agents.Applies,
	Errors: []fail.Code{UnknownApp},
	Run: func(ctx context.Context, d Deps, in StartIn, _ call.Progress) (Started, error) {
		// invariant: an agent starts in a terminal of its own on the VM's
		// multiplexer, the one the owner attaches to, its task typed as its
		// first prompt once it is ready.
		app := in.App
		if app == "" && len(d.Hooks) > 0 {
			app = d.Hooks[0].App()
		}
		hooks := d.hooksOf(app)
		if hooks == nil {
			return Started{}, fail.New(fail.NotFound, UnknownApp, string(in.App), nil)
		}
		terminal, err := d.Session.Open(ctx, app, in.Folder, hooks.Launch())
		if err != nil {
			return Started{}, err
		}
		if strings.TrimSpace(in.Task) != "" {
			if err := d.Session.Send(ctx, terminal, in.Task); err != nil {
				return Started{}, err
			}
		}
		return Started{App: app, Terminal: terminal, Folder: in.Folder, Task: in.Task}, nil
	},
}

type MergeIn struct {
	Session string
}

var MergePR = call.Call[Deps, MergeIn, agents.Agent]{
	// invariant: owner decision D2 (SPEC §D3), built on its recommendation:
	// the merge asks no confirmation, the gesture names the PR.
	Name: "agents.merge-pr", Where: call.AtTarget, Applies: agents.Applies,
	Errors: []fail.Code{UnknownSession, Ended, NoPR},
	Run: func(ctx context.Context, d Deps, in MergeIn, _ call.Progress) (agents.Agent, error) {
		// invariant: the agent's PR is merged in its folder with the VM's own
		// forge, and the agent is finished with it.
		a, err := d.running(ctx, in.Session)
		if err != nil {
			return a, err
		}
		if a.PR == nil {
			return a, fail.New(fail.Refused, NoPR, a.Session, nil)
		}
		if err := d.Forge.Merge(ctx, a.Folder, a.PR.Number); err != nil {
			return a, err
		}
		a = agents.Judge(a, agents.Event{Kind: agents.Ended, Session: a.Session, At: d.Now(), Reason: "merged PR #" + strconv.Itoa(a.PR.Number)})
		return a, d.Journal.Write(ctx, a)
	},
}
