package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

type (
	Sections = Deps[settings.Report]
	Overview = Reader[settings.Report]
)

func All(d Sections) []call.Handler {
	return []call.Handler{call.BindWatch(Read, New(d))}
}

type ReadIn struct {
	// value: every shared reading of the run dropped first, so the whole
	// machine is read again.
	Fresh bool
	// value: the other Macs asked too, their roll-up said in the verdict:
	// the estate's head.
	Elsewhere bool
}

type ReadOut struct {
	Summary overview.Summary
	// value: a VM whose create still runs, read from its Mac's record
	// since its guest does not answer yet: how far its create is, its
	// Summary empty.
	Creating *machines.Birth
}

var Read = call.Watch[*Overview, ReadIn, ReadOut]{
	Name: "overview.read", Section: section.Overview, Where: call.AtTarget, Applies: overview.Applies,
	Run: func(ctx context.Context, r *Overview, in ReadIn, _ call.Progress) (<-chan call.Update[ReadOut], error) {
		if in.Fresh {
			r.Forget()
		}
		return call.Updates(ctx, mapped(ctx, r.Compose(ctx, in.Elsewhere), func(s overview.Summary) ReadOut { return ReadOut{Summary: s} })), nil
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
