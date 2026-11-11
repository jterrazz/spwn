package confirm_test

import (
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/src/core/confirm"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/machine"
)

func TestEachGestureAsksWhatItsKindNeeds(t *testing.T) {
	for _, c := range []struct {
		gesture string
		action  confirm.Action
		kind    machine.Kind
		want    confirm.Level
	}{
		{"start a workspace", confirm.Start, machine.Workspace, confirm.None},
		{"restart a server", confirm.Restart, machine.Server, confirm.None},
		{"install an app", confirm.Install, machine.Computer, confirm.None},
		{"share a port", confirm.Share, machine.Workspace, confirm.None},
		{"stop a workspace", confirm.Stop, machine.Workspace, confirm.YesNo},
		{"stop a server", confirm.Stop, machine.Server, confirm.TypeName},
		{"delete a workspace", confirm.Delete, machine.Workspace, confirm.TypeName},
		{"delete a server", confirm.Delete, machine.Server, confirm.TypeName},
		{"remove an app", confirm.Remove, machine.Computer, confirm.YesNo},
		{"take back an account", confirm.TakeBack, machine.Computer, confirm.YesNo},
		{"forget a Mac", confirm.Forget, machine.Computer, confirm.YesNo},
		{"delete an image", confirm.DeleteImage, machine.Computer, confirm.YesNo},
		{"erase an app's data", confirm.Erase, machine.Computer, confirm.TypeName},
		{"build the image a create needs", confirm.BuildImage, machine.Workspace, confirm.YesNo},
		{"delete the ephemeral workspaces whose agent finished", confirm.Cleanup, machine.Computer, confirm.YesNo},
	} {
		if got := confirm.For(c.action, c.kind); got != c.want {
			t.Errorf("%s asks %s, want %s", c.gesture, got, c.want)
		}
	}
}

func TestATypedQuestionIsAnsweredByTheNameAlone(t *testing.T) {
	typed := confirm.For(confirm.Stop, machine.Server)
	if err := confirm.Require(typed, "infra", "infra"); err != nil {
		t.Errorf("--confirm=infra did not stop infra: %v", err)
	}
	for given, code := range map[string]fail.Code{"": confirm.Needed, confirm.Yes: confirm.Needed, "inf": confirm.Mismatch} {
		var e *fail.Error
		if err := confirm.Require(typed, "infra", given); !errors.As(err, &e) || e.Code != code || fail.ExitOf(err) != fail.ExitUsage {
			t.Errorf("stopping infra answered %q: %v, want %s, exit 2", given, err, code)
			continue
		}
		if e.Subject != "infra" || e.Detail["typed"] != given {
			t.Errorf("the refusal of %q does not name what was typed and what was asked: %+v", given, e)
		}
	}
}

func TestAYesNoQuestionIsAnsweredByYes(t *testing.T) {
	asks := confirm.For(confirm.Remove, machine.Computer)
	if err := confirm.Require(asks, "lazygit", confirm.Yes); err != nil {
		t.Errorf("--yes did not remove lazygit: %v", err)
	}
	for _, given := range []string{"", "no", "lazygit"} {
		var e *fail.Error
		if err := confirm.Require(asks, "lazygit", given); !errors.As(err, &e) || e.Code != confirm.YesNeeded || e.Type != fail.NeedsConfirmation {
			t.Errorf("removing lazygit answered %q: %v, want %s", given, err, confirm.YesNeeded)
		}
	}
	if err := confirm.Require(confirm.For(confirm.Install, machine.Computer), "jq", ""); err != nil {
		t.Errorf("installing jq asked: %v", err)
	}
}

func TestTheMenuBarAsksAtTheLevelItsFactsGrade(t *testing.T) {
	for _, c := range []struct {
		alert  string
		action confirm.Action
		kind   machine.Kind
		facts  []confirm.Fact
		want   confirm.Level
	}{
		{"alert.restart-server", confirm.Restart, machine.Server, []confirm.Fact{confirm.Healthy}, confirm.YesNo},
		{"a down server restarts at once", confirm.Restart, machine.Server, nil, confirm.None},
		{"alert.stop-server", confirm.Stop, machine.Server, []confirm.Fact{confirm.Healthy}, confirm.TypeName},
		{"alert.delete-server", confirm.Delete, machine.Server, nil, confirm.TypeName},
		{"alert.stop-vm", confirm.Stop, machine.Workspace, []confirm.Fact{confirm.Ephemeral}, confirm.YesNo},
		{"alert.stop-agent", confirm.StopAgent, machine.Workspace, nil, confirm.YesNo},
		{"alert.delete-vm.ephemeral", confirm.Delete, machine.Workspace, []confirm.Fact{confirm.Ephemeral}, confirm.YesNo},
		{"alert.delete-vm.persistent", confirm.Delete, machine.Workspace, nil, confirm.TypeName},
		{"alert.stop-all", confirm.StopEngine, machine.Computer, nil, confirm.YesNo},
		{"alert.restart-mac", confirm.RestartMac, machine.Computer, []confirm.Fact{confirm.HostsServer}, confirm.TypeName},
		{"a Mac hosting no server restarts on yes", confirm.RestartMac, machine.Computer, nil, confirm.YesNo},
		{"alert.forget-mac", confirm.Forget, machine.Computer, nil, confirm.YesNo},
		{"alert.take-back", confirm.TakeBack, machine.Workspace, nil, confirm.YesNo},
		{"alert.stop-share", confirm.StopSharing, machine.Workspace, nil, confirm.YesNo},
		{"alert.let-sleep", confirm.OffHere, machine.Computer, []confirm.Fact{confirm.HostsServer}, confirm.YesNo},
	} {
		if got := confirm.Of(c.action, c.kind, c.facts...); got != c.want {
			t.Errorf("%s asks %s, want %s", c.alert, got, c.want)
		}
	}
}
