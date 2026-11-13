package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/background"
	"github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

const (
	Unknown     fail.Code = "background.unknown"
	RootScope   fail.Code = "background.root-scope"
	IsTheEngine fail.Code = "background.is-the-engine"
)

type Deps struct {
	Jobs contract.Jobs
	// value: the job manager's domain of the owner's own jobs.
	Domain contract.Domain
	// value: the owner's home and the system root, so a path reads as the
	// owner types it.
	Home, Root string
	Kind       machine.Kind
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.Bind(List, d), call.Bind(Remove, d)}
}

func tasks(ctx context.Context, d Deps) ([]background.Job, []background.Task, error) {
	jobs, err := d.Jobs.List(ctx)
	if err != nil {
		return nil, nil, err
	}
	return jobs, background.Judge(jobs, d.Root, d.Home), nil
}
