package initcmd

import (
	"context"
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type recording struct{ invoked int }

func (r *recording) Invoke(context.Context, machine.Ref, string, any, any, call.Progress) error {
	r.invoked++
	return nil
}

func (r *recording) Watch(context.Context, machine.Ref, string, any, call.Progress) (<-chan call.Update[any], error) {
	return nil, errors.New("init is invoked, not watched")
}

func TestAMachinefileTheApplyWouldRefuseStopsInitBeforeAnyStep(t *testing.T) {
	calls := &recording{}
	refusal := errors.New("no Machinefile named nonexistent")
	d := Deps{
		Resolve: func(context.Context, string) error { return refusal },
		Apply:   func(context.Context, string, kit.IO) error { t.Error("apply ran after a refused resolve"); return nil },
	}
	leaf := Leaf(d, kit.Env{Calls: calls}, "")
	err := leaf.Run(t.Context(), &Input{Name: "macbook", Machinefile: "nonexistent"}, kit.IO{})
	if !errors.Is(err, refusal) {
		t.Errorf("init = %v; want the resolve's refusal", err)
	}
	if calls.invoked != 0 {
		t.Errorf("%d setup calls ran; want none before the Machinefile resolves", calls.invoked)
	}
}
