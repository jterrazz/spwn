package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/disk"
)

type Deps struct {
	Disk Disk
	Kind machine.Kind
}

type Disk interface {
	Watch(ctx context.Context) <-chan disk.Survey
	Volume(ctx context.Context) (total, free int64, err error)
	Free(ctx context.Context, names []string) ([]disk.Outcome, error)
	CheckRunning(ctx context.Context, name string) error
	Erase(ctx context.Context, name string) (disk.Outcome, error)
	Move(ctx context.Context, folder string) (disk.Outcome, error)
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.BindWatch(Read, d), call.Bind(Free, d), call.Bind(Erase, d), call.Bind(Move, d)}
}
