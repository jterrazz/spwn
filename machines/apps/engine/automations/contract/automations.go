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
	// invariant: the automation a ref names, a name or `group/name`; a ref none declares is automations.ErrNotFound, and a name two groups declare is a *automations.NotUniqueError.
	Automation(ctx context.Context, project, name string) (automations.Automation, error)
	// invariant: the runs written for the project, oldest first, those of
	// the one automation a ref names (refused as Automation refuses it), and
	// only the newest last of them.
	Log(ctx context.Context, project, name string, last int) ([]automations.Run, error)
}
