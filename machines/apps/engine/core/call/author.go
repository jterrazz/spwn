package call

import (
	"context"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/journal"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

type byKey struct{}

func WithBy(ctx context.Context, by string) context.Context {
	// value: who a call is done for, a session's name or an engine job's,
	// carried to every frame it sends; empty is the owner.
	return context.WithValue(ctx, byKey{}, by)
}

func By(ctx context.Context) string {
	by, _ := ctx.Value(byKey{}).(string)
	return by
}

func Session(name string) bool {
	// invariant: a session's name is 1 to 40 lowercase letters, digits and
	// "-", and never the owner's own word.
	return len(name) >= 1 && len(name) <= 40 && name != journal.ByYou && strings.IndexFunc(name, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-'
	}) < 0
}

func FrameBy(name, by string) error {
	// invariant: a frame names its author as a session or not at all: an
	// engine job's own name never comes from another process.
	if by == "" || Session(by) {
		return nil
	}
	return fail.New(fail.Refused, BadInput, name, map[string]string{"in": "by"})
}

func Author(ctx context.Context, name string) error {
	// invariant: a gesture is done by the owner, a session or one of the
	// engine's jobs; any other author refuses it before it runs.
	switch by := By(ctx); by {
	case "", journal.ByYou, journal.ByNight, journal.ByStation:
		return nil
	default:
		if Session(by) {
			return nil
		}
	}
	return fail.New(fail.Refused, BadInput, name, map[string]string{"in": "by"})
}

type Keeper func(ctx context.Context, target machine.Ref, m Meta, in, out any, err error)

func (k Keeper) Keep(ctx context.Context, on machine.Profile, target machine.Ref, m Meta, in, out any, err error) {
	// invariant: a gesture that ended is kept on a Computer, handed the
	// machine it ran on, when it declares its lines and no kept gesture
	// runs it: the outer gesture's lines say what was done, so the owner
	// reads one line per thing they did.
	if k == nil || on.Kind != machine.Computer || m.Journal == nil || ctx.Value(insideKey{}) != nil {
		return
	}
	k(ctx, target, m, in, out, err)
}

type insideKey struct{}

func inside(ctx context.Context, m Meta) context.Context {
	// value: the context a gesture that declares its lines runs on, marked
	// so a gesture it runs in turn is kept by none.
	if m.Journal == nil {
		return ctx
	}
	return context.WithValue(ctx, insideKey{}, m.Name)
}
