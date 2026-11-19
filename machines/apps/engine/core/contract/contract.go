package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

type Self interface {
	Read(ctx context.Context) (machine.Ref, machine.Kind, machine.System, machine.Hardware, error)
}

type Preferences interface {
	// invariant: untyped by group, so core never knows a family's record;
	// each family types its own group in its contract package.
	// invariant: a group never written leaves v as it was handed.
	Read(group string, v any) error
	// invariant: writing one group keeps every other as it was; an empty
	// value removes its group.
	Write(group string, v any) error
}
