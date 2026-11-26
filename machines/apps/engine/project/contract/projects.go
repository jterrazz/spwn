package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/project"
)

type Projects interface {
	// invariant: every method reads the project as its files declare it,
	// on this machine, and writes nothing outside the folder of the project
	// it makes or applies; folder and root are absolute paths.

	// invariant: the root of the project folder is in, its spwn.yaml found
	// from folder up; project.ErrNotFound when none is.
	Find(ctx context.Context, folder string) (string, error)
	// invariant: the project at root, its agents by name.
	Show(ctx context.Context, root string) (project.Project, error)
	// invariant: a new folder, folder/name, made a project named name with
	// its starter agent on harness; name is valid and harness one of the
	// catalog's. A folder/name that exists is project.ErrExists, and
	// nothing is written.
	Create(ctx context.Context, folder, name, harness string) (project.Project, error)
	// invariant: what the project's rules find in its declaration, as
	// project.PlanOf orders it, with no Changes; it changes nothing.
	Plan(ctx context.Context, root string) (project.Plan, error)
	// invariant: how what was installed for the project differs from what
	// it declares; it changes nothing.
	Drift(ctx context.Context, root string) (project.Drift, error)
	// invariant: the Render hook: the files of agent's run folder, as the
	// harness it declares renders them (the agent's own, else the
	// project's, else the catalog's first), in memory; it writes nothing
	// and reads only the project's declaration. Plan and apply both read
	// it, so what a plan lists is what an apply writes.
	Render(ctx context.Context, root, agent string) ([]project.File, error)
	// invariant: what apply last wrote under root, and the digest of what
	// each of paths holds now, a path from root; a path that holds nothing
	// has no digest.
	Applied(ctx context.Context, root string, paths []string) (record, disk project.Record, err error)
	// invariant: each change made true under root, its folders made as
	// needed, then record kept as what apply wrote; nothing outside root
	// is written, and no path a change does not name is touched.
	Write(ctx context.Context, root string, changes []project.Change, record project.Record) error
}
