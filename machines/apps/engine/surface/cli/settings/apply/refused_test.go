package settingsapply

import (
	"context"
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	pagesettings "github.com/jterrazz/studio-station/apps/engine/pages/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	settingscalls "github.com/jterrazz/studio-station/apps/engine/settings/calls"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

func refusal() string { return "fetch https://bot:hunter2@proxy.example/x refused" }

func TestAMachineAnApplyFailedOnSaysWhyWithItsTokenWithheld(t *testing.T) {
	var a pagesettings.MachineApply
	if !refusedOn(&a, errors.New(refusal())) || a.Err != "fetch https://withheld@proxy.example/x refused" {
		t.Errorf("apply on a machine = %+v; want a failure with its credential withheld", a)
	}
}

func TestAMachineNoSettingLivesOnSaysWhyWithItsTokenWithheld(t *testing.T) {
	var a pagesettings.MachineApply
	err := &fail.Error{Type: fail.NotForThisMachine, Message: refusal()}
	if refusedOn(&a, err) || a.Skipped != "fetch https://withheld@proxy.example/x refused" {
		t.Errorf("apply on a machine = %+v; want it skipped with its credential withheld", a)
	}
}

type elsewhere struct{}

func (elsewhere) Invoke(_ context.Context, _ machine.Ref, _ string, _, _ any, _ call.Progress) error {
	return &fail.Error{Type: fail.NotForThisMachine, Code: settingscalls.NotForThisMachine, Subject: "dock", Message: refusal()}
}

func (elsewhere) Watch(context.Context, machine.Ref, string, any, call.Progress) (<-chan call.Update[any], error) {
	return nil, errors.New("apply is invoked, not watched")
}

func TestASettingThatLivesElsewhereSaysWhyWithItsTokenWithheld(t *testing.T) {
	var a pagesettings.MachineApply
	if err := applyLiving(t.Context(), kit.Env{Calls: elsewhere{}}, settings.LivesOn{}, []string{"dock"}, kit.IO{}, &a); err != nil {
		t.Fatalf("apply failed with %v; want the setting said elsewhere", err)
	}
	if len(a.Elsewhere) != 1 || a.Elsewhere[0].Why != "fetch https://withheld@proxy.example/x refused" {
		t.Errorf("elsewhere = %+v; want its credential withheld", a.Elsewhere)
	}
}
