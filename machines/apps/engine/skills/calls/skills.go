package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/skills"
)

// value: every call's Project is the root of the project its skills belong
// to, an absolute path on the target; empty for the user's own skills.

type ListIn struct {
	Project string `json:"project,omitempty"`
}

var List = call.Call[Deps, ListIn, []skills.Skill]{
	Name: "skills.list", Where: call.AtTarget, Applies: machine.Anywhere,
	Run: func(ctx context.Context, d Deps, in ListIn, _ call.Progress) ([]skills.Skill, error) {
		return d.Files.List(ctx, d.root(in.Project))
	},
}

type ShowIn struct {
	Project string `json:"project,omitempty"`
	Skill   string
}

var Show = call.Call[Deps, ShowIn, skills.Skill]{
	Name: "skills.show", Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{NotFound},
	Run: func(ctx context.Context, d Deps, in ShowIn, _ call.Progress) (skills.Skill, error) {
		s, err := d.Files.Show(ctx, d.root(in.Project), in.Skill)
		return s, coded(err, in.Skill)
	},
}

type CreateIn struct {
	Project string `json:"project,omitempty"`
	Skill   string
}

var Create = call.Call[Deps, CreateIn, skills.Skill]{
	Name: "skills.create", Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{BadName, Exists},
	Run: func(ctx context.Context, d Deps, in CreateIn, _ call.Progress) (skills.Skill, error) {
		if !skills.ValidName(in.Skill) {
			return skills.Skill{}, fail.New(fail.Refused, BadName, in.Skill, nil)
		}
		s, err := d.Files.Create(ctx, d.root(in.Project), in.Skill)
		return s, coded(err, in.Skill)
	},
}

type DeleteIn struct {
	Project string `json:"project,omitempty"`
	Skill   string
	Confirm string
}

var Delete = call.Call[Deps, DeleteIn, skills.Skill]{
	Name: "skills.delete", Where: call.AtTarget, Applies: machine.Anywhere, Confirm: confirm.DeleteSkill,
	Errors: []fail.Code{NotFound, confirm.Needed, confirm.Mismatch},
	Run: func(ctx context.Context, d Deps, in DeleteIn, _ call.Progress) (skills.Skill, error) {
		// invariant: a skill's file is gone for good, so its name is typed
		// first; a name no skill has is refused before the question.
		s, err := d.Files.Show(ctx, d.root(in.Project), in.Skill)
		if err != nil {
			return skills.Skill{}, coded(err, in.Skill)
		}
		s.Body = ""
		if err := confirm.Require(confirm.Of(confirm.DeleteSkill, d.Kind), in.Skill, in.Confirm); err != nil {
			return s, err
		}
		return s, coded(d.Files.Delete(ctx, d.root(in.Project), in.Skill), in.Skill)
	},
}
