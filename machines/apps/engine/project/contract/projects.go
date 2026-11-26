package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/project"
)

type Projects interface {
	// invariant: every method reads the project as its files declare it,
	// on this machine, and writes nothing outside the folder of the project
	// it makes; folder and root are absolute paths.

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
	// project.PlanOf orders it; it changes nothing.
	Plan(ctx context.Context, root string) (project.Plan, error)
	// invariant: how what was installed for the project differs from what
	// it declares; it changes nothing.
	Drift(ctx context.Context, root string) (project.Drift, error)
}
