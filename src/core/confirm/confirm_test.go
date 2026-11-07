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
		{"install an app", confirm.Install, machine.Mac, confirm.None},
		{"share a port", confirm.Share, machine.Workspace, confirm.None},
		{"stop a workspace", confirm.Stop, machine.Workspace, confirm.YesNo},
		{"stop a server", confirm.Stop, machine.Server, confirm.TypeName},
		{"delete a workspace", confirm.Delete, machine.Workspace, confirm.TypeName},
		{"delete a server", confirm.Delete, machine.Server, confirm.TypeName},
		{"remove an app", confirm.Remove, machine.Mac, confirm.YesNo},
		{"take back an account", confirm.TakeBack, machine.Mac, confirm.YesNo},
		{"forget a Mac", confirm.Forget, machine.Mac, confirm.YesNo},
		{"delete an image", confirm.DeleteImage, machine.Mac, confirm.YesNo},
		{"erase an app's data", confirm.Erase, machine.Mac, confirm.TypeName},
		{"build the image a create needs", confirm.BuildImage, machine.Workspace, confirm.YesNo},
		{"delete the throwaways whose agent finished", confirm.Cleanup, machine.Mac, confirm.YesNo},
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
	asks := confirm.For(confirm.Remove, machine.Mac)
	if err := confirm.Require(asks, "lazygit", confirm.Yes); err != nil {
		t.Errorf("--yes did not remove lazygit: %v", err)
	}
	for _, given := range []string{"", "no", "lazygit"} {
		var e *fail.Error
		if err := confirm.Require(asks, "lazygit", given); !errors.As(err, &e) || e.Code != confirm.YesNeeded || e.Type != fail.NeedsConfirmation {
			t.Errorf("removing lazygit answered %q: %v, want %s", given, err, confirm.YesNeeded)
		}
	}
	if err := confirm.Require(confirm.For(confirm.Install, machine.Mac), "jq", ""); err != nil {
		t.Errorf("installing jq asked: %v", err)
	}
}
