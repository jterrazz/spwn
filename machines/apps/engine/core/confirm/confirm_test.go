package confirm_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
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
		{"machines.restart of a server", confirm.RestartServer, machine.Server, confirm.TypeName},
		{"install an app", confirm.Install, machine.Computer, confirm.None},
		{"share a port", confirm.Share, machine.Workspace, confirm.None},
		{"stop a workspace", confirm.Stop, machine.Workspace, confirm.YesNo},
		{"stop a server", confirm.Stop, machine.Server, confirm.TypeName},
		{"delete a workspace", confirm.Delete, machine.Workspace, confirm.TypeName},
		{"delete a server", confirm.Delete, machine.Server, confirm.TypeName},
		{"remove an app", confirm.Remove, machine.Computer, confirm.YesNo},
		{"forget a Mac", confirm.Forget, machine.Computer, confirm.YesNo},
		{"delete an image", confirm.DeleteImage, machine.Computer, confirm.YesNo},
		{"erase an app's data", confirm.Erase, machine.Computer, confirm.TypeName},
		{"build the image a create needs", confirm.BuildImage, machine.Workspace, confirm.YesNo},
		{"delete the stopped ephemeral workspaces", confirm.Cleanup, machine.Computer, confirm.YesNo},
		{"free what nothing uses on a Mac", confirm.FreeUnused, machine.Computer, confirm.TypeName},
		{"move the apps' files out of the home root", confirm.Relocate, machine.Computer, confirm.YesNo},
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
		{"alert.restart-server", confirm.RestartServer, machine.Server, nil, confirm.TypeName},
		{"alert.stop-server", confirm.Stop, machine.Server, nil, confirm.TypeName},
		{"alert.delete-server", confirm.Delete, machine.Server, nil, confirm.TypeName},
		{"alert.stop-vm", confirm.Stop, machine.Workspace, []confirm.Fact{confirm.Ephemeral}, confirm.YesNo},
		{"alert.delete-vm.ephemeral", confirm.Delete, machine.Workspace, []confirm.Fact{confirm.Ephemeral}, confirm.YesNo},
		{"alert.delete-vm.persistent", confirm.Delete, machine.Workspace, nil, confirm.TypeName},
		{"alert.stop-all", confirm.StopEngine, machine.Computer, nil, confirm.YesNo},
		{"alert.restart-mac", confirm.RestartMac, machine.Computer, []confirm.Fact{confirm.HostsServer}, confirm.TypeName},
		{"a Mac hosting no server restarts on yes", confirm.RestartMac, machine.Computer, nil, confirm.YesNo},
		{"alert.forget-mac", confirm.Forget, machine.Computer, nil, confirm.YesNo},
		{"alert.stop-share", confirm.StopSharing, machine.Workspace, nil, confirm.YesNo},
		{"alert.let-sleep", confirm.OffHere, machine.Computer, []confirm.Fact{confirm.HostsServer}, confirm.YesNo},
	} {
		if got := confirm.Of(c.action, c.kind, c.facts...); got != c.want {
			t.Errorf("%s asks %s, want %s", c.alert, got, c.want)
		}
	}
}

func TestEveryConfirmationSaysWhatItsGestureDoes(t *testing.T) {
	for _, a := range []confirm.Action{confirm.Install, confirm.Share, confirm.Start, confirm.Restart, confirm.Edit, confirm.Remove,
		confirm.StopSharing, confirm.SignOut, confirm.RemovePassword, confirm.RemoveGrant, confirm.Undeclare, confirm.OffHere, confirm.Overwrite,
		confirm.Forget, confirm.DeleteImage, confirm.RestartMac, confirm.RestartServer, confirm.StopEngine, confirm.Stop, confirm.Delete, confirm.Erase,
		confirm.BuildImage, confirm.Cleanup, confirm.FreeUnused, confirm.Relocate} {
		if c := a.Consequence(); c == "" || !strings.HasSuffix(c, ".") {
			t.Errorf("%s says %q: every confirmation says what it does, in one sentence", a, c)
		}
	}
	if got := confirm.RestartServer.Consequence(); got != "It is a server: what it serves stops while it restarts." {
		t.Errorf("a server's restart says %q: it says it is a server and what stops while it restarts", got)
	}
}
