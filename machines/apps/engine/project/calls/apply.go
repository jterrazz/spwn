package calls

import (
	"context"
	"maps"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/project"
)

// value: what an apply of the project at root would change, and the record
// it then keeps, from each agent's Render and what the disk holds.
func pending(ctx context.Context, d Deps, root string) (changes []project.Change, record, next project.Record, err error) {
	p, err := d.Projects.Show(ctx, root)
	if err != nil {
		return nil, nil, nil, err
	}
	rendered := map[string][]project.File{}
	for _, agent := range p.Agents {
		if rendered[agent], err = d.Projects.Render(ctx, root, agent); err != nil {
			return nil, nil, nil, err
		}
	}
	if record, _, err = d.Projects.Applied(ctx, root, nil); err != nil {
		return nil, nil, nil, err
	}
	_, disk, err := d.Projects.Applied(ctx, root, project.Paths(rendered, record))
	if err != nil {
		return nil, nil, nil, err
	}
	changes, next, err = project.ChangesOf(rendered, record, disk)
	return changes, record, next, err
}

type ApplyIn struct {
	Folder  string
	Confirm string
}

var Apply = call.Call[Deps, ApplyIn, project.Applied]{
	// value: the project's declared state made true in its run folders:
	// what its plan lists, and nothing else.
	Name: "project.apply", Where: call.InProcess, Applies: machine.Anywhere, Confirm: confirm.Overwrite,
	Errors: []fail.Code{NotFound, Invalid, LinkInTheWay, NotAFile, NotAFolder, NotWritten, confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in ApplyIn, _ call.Progress) (project.Applied, error) {
		// invariant: a plan with an error writes nothing; an apply that
		// only adds asks nothing; one that overwrites or removes a file
		// asks once, naming each, before anything is written; one that
		// changes nothing writes nothing.
		root, err := d.Projects.Find(ctx, in.Folder)
		if err != nil {
			return project.Applied{}, coded(err, in.Folder)
		}
		plan, err := d.Projects.Plan(ctx, root)
		if err != nil {
			return project.Applied{}, coded(err, root)
		}
		out := project.Applied{Root: root, Manifest: plan.Manifest}
		if refused := Refused(plan); refused != nil {
			return out, refused
		}
		changes, record, next, err := pending(ctx, d, root)
		if err != nil {
			return out, coded(err, root)
		}
		out.Changes = changes
		if asked := project.Asked(changes); len(asked) > 0 {
			if refused := confirm.Require(confirm.Of(confirm.Overwrite, machine.Computer), project.Goes(asked), in.Confirm); refused != nil {
				return out, refused
			}
		}
		if len(changes) == 0 && maps.Equal(record, next) {
			return out, nil
		}
		return out, coded(d.Projects.Write(ctx, root, changes, next), root)
	},
}
