package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/skills"
)

type Files interface {
	// invariant: every method acts on the skill files of one project, its
	// root an absolute path on this machine; an empty project is the user's
	// own skills. A name no skill has, or that no file can be named, is
	// skills.ErrNotFound; a name taken, skills.ErrExists.

	// invariant: every skill of the project by name, its Body empty.
	List(ctx context.Context, project string) ([]skills.Skill, error)
	// invariant: the skill with its whole file read.
	Show(ctx context.Context, project, name string) (skills.Skill, error)
	// invariant: a new file holding skills.Template(name); name is valid.
	Create(ctx context.Context, project, name string) (skills.Skill, error)
	// invariant: the skill's file is gone.
	Delete(ctx context.Context, project, name string) error
}
