package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/automations"
)

type Automations interface {
	// invariant: the automations are those the project declares, its root
	// an absolute path on this machine; no project, or a root with no
	// manifest, declares none. Nothing here schedules, fires or changes
	// one.

	// invariant: every automation by name, with what its runs say.
	List(ctx context.Context, project string) ([]automations.Automation, error)
	// invariant: the automation of the name; a name none declares is automations.ErrNotFound.
	Automation(ctx context.Context, project, name string) (automations.Automation, error)
	// invariant: the runs written for the project, oldest first, those of
	// one automation when a name is given (a name none declares is
	// automations.ErrNotFound), and only the newest last of them.
	Log(ctx context.Context, project, name string, last int) ([]automations.Run, error)
}
