package calls

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	projectcalls "github.com/jterrazz/studio-station/apps/engine/project/calls"
)

// value: every call's Project is the root of the project its agents belong
// to, an absolute path on the target; empty for the user's own agents.

type ListIn struct {
	Project string `json:"project,omitempty"`
}

var List = call.Call[Deps, ListIn, []agents.Agent]{
	Name: "agents.list", Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{projectcalls.OldLayout, projectcalls.FoldersInvalid},
	Run: func(ctx context.Context, d Deps, in ListIn, _ call.Progress) ([]agents.Agent, error) {
		all, err := d.Minds.List(ctx, d.root(in.Project))
		return all, coded(err, "")
	},
}

type ShowIn struct {
	Project string `json:"project,omitempty"`
	Agent   string
}

var Show = call.Call[Deps, ShowIn, agents.Mind]{
	Name: "agents.show", Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{NotFound, projectcalls.OldLayout, projectcalls.FoldersInvalid},
	Run: func(ctx context.Context, d Deps, in ShowIn, _ call.Progress) (agents.Mind, error) {
		m, err := d.Minds.Mind(ctx, d.root(in.Project), in.Agent)
		return m, coded(err, in.Agent)
	},
}

type CreateIn struct {
	Project string `json:"project,omitempty"`
	Agent   string
	Team    *string `json:",omitempty"`
	Harness *string `json:",omitempty"`
}

var Create = call.Call[Deps, CreateIn, agents.Mind]{
	Name: "agents.create", Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{BadName, Exists, UnknownHarness, projectcalls.OldLayout, projectcalls.FoldersInvalid},
	Run: func(ctx context.Context, d Deps, in CreateIn, _ call.Progress) (agents.Mind, error) {
		e := agents.Edit{Team: in.Team, Harness: in.Harness}
		if err := named(in.Agent); err != nil {
			return agents.Mind{}, err
		}
		if err := checked(e); err != nil {
			return agents.Mind{}, err
		}
		m, err := d.Minds.Create(ctx, d.root(in.Project), in.Agent, e)
		return m, coded(err, in.Agent)
	},
}

type EditIn struct {
	Project string `json:"project,omitempty"`
	Agent   string
	Team    *string `json:",omitempty"`
	Harness *string `json:",omitempty"`
}

var Edit = call.Call[Deps, EditIn, agents.Agent]{
	Name: "agents.edit", Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{NothingAsked, NotFound, UnknownHarness, projectcalls.OldLayout, projectcalls.FoldersInvalid},
	Run: func(ctx context.Context, d Deps, in EditIn, _ call.Progress) (agents.Agent, error) {
		e := agents.Edit{Team: in.Team, Harness: in.Harness}
		if e == (agents.Edit{}) {
			return agents.Agent{}, fail.New(fail.Refused, NothingAsked, in.Agent, nil)
		}
		if err := checked(e); err != nil {
			return agents.Agent{}, err
		}
		a, err := d.Minds.Edit(ctx, d.root(in.Project), in.Agent, e)
		return a, coded(err, in.Agent)
	},
}

type DeleteIn struct {
	Project string `json:"project,omitempty"`
	Agent   string
	Confirm string
}

var Delete = call.Call[Deps, DeleteIn, agents.Agent]{
	Name: "agents.delete", Where: call.AtTarget, Applies: machine.Anywhere, Confirm: confirm.DeleteAgent,
	Errors: []fail.Code{NotFound, confirm.Needed, confirm.Mismatch, projectcalls.OldLayout, projectcalls.FoldersInvalid},
	Run: func(ctx context.Context, d Deps, in DeleteIn, _ call.Progress) (agents.Agent, error) {
		// invariant: an agent's Mind is gone for good, so its name is typed
		// first; a name no agent has is refused before the question.
		m, err := d.Minds.Mind(ctx, d.root(in.Project), in.Agent)
		if err != nil {
			return agents.Agent{}, coded(err, in.Agent)
		}
		if err := confirm.Require(confirm.Of(confirm.DeleteAgent, d.Kind), in.Agent, in.Confirm); err != nil {
			return m.Agent, err
		}
		return m.Agent, coded(d.Minds.Delete(ctx, d.root(in.Project), in.Agent), in.Agent)
	},
}

type ForkIn struct {
	Project string `json:"project,omitempty"`
	Agent   string
	As      string
}

var Fork = call.Call[Deps, ForkIn, agents.Forked]{
	Name: "agents.fork", Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{NotFound, BadName, Exists, projectcalls.OldLayout, projectcalls.FoldersInvalid},
	Run: func(ctx context.Context, d Deps, in ForkIn, _ call.Progress) (agents.Forked, error) {
		if err := named(in.As); err != nil {
			return agents.Forked{}, err
		}
		f, err := d.Minds.Fork(ctx, d.root(in.Project), in.Agent, in.As)
		if err != nil && !errors.Is(err, agents.ErrNotFound) {
			return f, coded(err, in.As)
		}
		return f, coded(err, in.Agent)
	},
}

type ExportIn struct {
	Project string `json:"project,omitempty"`
	Agent   string
	// value: the folder the archive is written into, an absolute path on the target.
	To string
}

var Export = call.Call[Deps, ExportIn, agents.Exported]{
	Name: "agents.export", Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{NotFound, projectcalls.OldLayout, projectcalls.FoldersInvalid},
	Run: func(ctx context.Context, d Deps, in ExportIn, _ call.Progress) (agents.Exported, error) {
		x, err := d.Minds.Export(ctx, d.root(in.Project), in.Agent, in.To)
		return x, coded(err, in.Agent)
	},
}

type ImportIn struct {
	Project string `json:"project,omitempty"`
	// value: the archive read, an absolute path on the target.
	Archive string
	// value: the name the agent takes; empty for the archive's own.
	As string `json:",omitempty"`
}

var Import = call.Call[Deps, ImportIn, agents.Mind]{
	Name: "agents.import", Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{BadName, Exists, BadArchive, projectcalls.OldLayout, projectcalls.FoldersInvalid},
	Run: func(ctx context.Context, d Deps, in ImportIn, _ call.Progress) (agents.Mind, error) {
		as := in.As
		if as == "" {
			as = agents.ArchiveName(in.Archive)
		}
		if err := named(as); err != nil {
			return agents.Mind{}, err
		}
		m, err := d.Minds.Import(ctx, d.root(in.Project), in.Archive, as)
		if errors.Is(err, agents.ErrBadArchive) {
			return m, coded(err, in.Archive)
		}
		return m, coded(err, as)
	},
}

type DreamIn struct {
	Project string `json:"project,omitempty"`
	Agent   string
}

var Dream = call.Call[Deps, DreamIn, agents.Dreamed]{
	Name: "agents.dream", Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{NotFound, projectcalls.OldLayout, projectcalls.FoldersInvalid},
	Run: func(ctx context.Context, d Deps, in DreamIn, _ call.Progress) (agents.Dreamed, error) {
		r, err := d.Minds.Dream(ctx, d.root(in.Project), in.Agent)
		return r, coded(err, in.Agent)
	},
}

type SleepIn struct {
	Project string `json:"project,omitempty"`
	Agent   string
}

var Sleep = call.Call[Deps, SleepIn, agents.Slept]{
	Name: "agents.sleep", Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{NotFound, projectcalls.OldLayout, projectcalls.FoldersInvalid},
	Run: func(ctx context.Context, d Deps, in SleepIn, _ call.Progress) (agents.Slept, error) {
		r, err := d.Minds.Sleep(ctx, d.root(in.Project), in.Agent)
		return r, coded(err, in.Agent)
	},
}
