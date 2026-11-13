package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/disk"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

type (
	Snapshot = overview.Snapshot[apps.Report, settings.Report, disk.MisplacedReport]
	Sections = Deps[apps.Report, settings.Report, disk.MisplacedReport]
	Overview = Reader[apps.Report, settings.Report, disk.MisplacedReport]
)

func All(d Sections) []call.Handler {
	return []call.Handler{call.BindWatch(Read, New(d))}
}

type ReadIn struct {
	// value: every shared reading of the run dropped first, so the whole
	// machine is read again.
	Fresh bool
	// value: the v2 reading instead, which only the status alias's
	// station.status/v1 still draws.
	V2 bool
}

type ReadOut struct {
	Summary overview.Summary
	V2      *Snapshot
}

var Read = call.Watch[*Overview, ReadIn, ReadOut]{
	Name: "overview.read", Section: section.Overview, Where: call.AtTarget, Applies: overview.Applies,
	Run: func(ctx context.Context, r *Overview, in ReadIn, _ call.Progress) (<-chan call.Update[ReadOut], error) {
		if in.Fresh {
			r.Forget()
		}
		if !in.V2 {
			return call.Updates(ctx, mapped(ctx, r.Compose(ctx), func(s overview.Summary) ReadOut { return ReadOut{Summary: s} })), nil
		}
		return call.Updates(ctx, mapped(ctx, r.Watch(ctx), func(s Snapshot) ReadOut { return ReadOut{V2: &s} })), nil
	},
}

func mapped[A, B any](ctx context.Context, in <-chan A, f func(A) B) <-chan B {
	out := make(chan B)
	go func() {
		defer close(out)
		for v := range in {
			select {
			case out <- f(v):
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}
