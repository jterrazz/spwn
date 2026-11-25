package confirm_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

func TestEachGestureAsksWhatItCanLose(t *testing.T) {
	running := []confirm.Fact{confirm.Ephemeral, confirm.Running}
	for _, c := range []struct {
		gesture string
		action  confirm.Action
		kind    machine.Kind
		facts   []confirm.Fact
		want    confirm.Level
	}{
		{"start a workspace", confirm.Start, machine.Workspace, nil, confirm.None},
		{"restart a workspace", confirm.Restart, machine.Workspace, nil, confirm.None},
		{"restart an up server", confirm.RestartServer, machine.Server, []confirm.Fact{confirm.Running}, confirm.YesNo},
		{"restart a down server", confirm.RestartServer, machine.Server, []confirm.Fact{confirm.Running, confirm.Down}, confirm.None},
		{"install an app", confirm.Install, machine.Computer, nil, confirm.None},
		{"share a port", confirm.Share, machine.Workspace, nil, confirm.None},
		{"stop sharing a port", confirm.StopSharing, machine.Workspace, nil, confirm.None},
		{"stop a workspace", confirm.Stop, machine.Workspace, running, confirm.None},
		{"stop a server", confirm.Stop, machine.Server, nil, confirm.YesNo},
		{"stop the engine", confirm.StopEngine, machine.Computer, nil, confirm.YesNo},
		{"restart a Mac", confirm.RestartMac, machine.Computer, nil, confirm.YesNo},
		{"delete a running ephemeral workspace", confirm.Delete, machine.Workspace, running, confirm.TypeName},
		{"delete a held ephemeral workspace", confirm.Delete, machine.Workspace, []confirm.Fact{confirm.Ephemeral, confirm.Held}, confirm.TypeName},
		{"delete a stopped ephemeral workspace nobody holds", confirm.Delete, machine.Workspace, []confirm.Fact{confirm.Ephemeral}, confirm.None},
		{"delete a stopped persistent workspace", confirm.Delete, machine.Workspace, nil, confirm.TypeName},
		{"delete a server", confirm.Delete, machine.Server, nil, confirm.TypeName},
		{"remove an app", confirm.Remove, machine.Computer, nil, confirm.YesNo},
		{"forget a Mac", confirm.Forget, machine.Computer, nil, confirm.YesNo},
		{"delete an image", confirm.DeleteImage, machine.Computer, nil, confirm.YesNo},
		{"erase an app's data", confirm.Erase, machine.Computer, nil, confirm.TypeName},
		{"build the image a create needs", confirm.BuildImage, machine.Workspace, nil, confirm.None},
		{"delete the stopped ephemeral workspaces", confirm.Cleanup, machine.Computer, nil, confirm.YesNo},
		{"free what nothing uses on a Mac", confirm.FreeUnused, machine.Computer, nil, confirm.YesNo},
		{"move the apps' files out of the home root", confirm.Relocate, machine.Computer, nil, confirm.None},
		{"turn a setting off here", confirm.OffHere, machine.Computer, nil, confirm.None},
		{"let a Mac hosting a server sleep", confirm.OffHere, machine.Computer, []confirm.Fact{confirm.HostsServer}, confirm.YesNo},
	} {
		if got := confirm.Of(c.action, c.kind, c.facts...); got != c.want {
			t.Errorf("%s asks %s, want %s", c.gesture, got, c.want)
		}
	}
}

func TestATypedQuestionIsAnsweredByTheNameAlone(t *testing.T) {
	typed := confirm.Of(confirm.Erase, machine.Computer)
	if err := confirm.Require(typed, "ollama", "ollama"); err != nil {
		t.Errorf("--confirm=ollama did not erase ollama: %v", err)
	}
	for given, code := range map[string]fail.Code{"": confirm.Needed, confirm.Yes: confirm.Needed, "oll": confirm.Mismatch} {
		var e *fail.Error
		if err := confirm.Require(typed, "ollama", given); !errors.As(err, &e) || e.Code != code || fail.ExitOf(err) != fail.ExitUsage {
			t.Errorf("erasing ollama answered %q: %v, want %s, exit 2", given, err, code)
			continue
		}
		if e.Subject != "ollama" || e.Detail["typed"] != given {
			t.Errorf("the refusal of %q does not name what was typed and what was asked: %+v", given, e)
		}
	}
}

func TestAYesNoQuestionIsAnsweredByYesOrByTheNameItAsksAbout(t *testing.T) {
	asks := confirm.Of(confirm.Remove, machine.Computer)
	for _, given := range []string{confirm.Yes, "lazygit"} {
		if err := confirm.Require(asks, "lazygit", given); err != nil {
			t.Errorf("%q did not remove lazygit: %v", given, err)
		}
	}
	for _, given := range []string{"", "no", "lazy"} {
		var e *fail.Error
		if err := confirm.Require(asks, "lazygit", given); !errors.As(err, &e) || e.Code != confirm.YesNeeded || e.Type != fail.NeedsConfirmation {
			t.Errorf("removing lazygit answered %q: %v, want %s", given, err, confirm.YesNeeded)
		}
	}
	if err := confirm.Require(confirm.Of(confirm.Install, machine.Computer), "jq", ""); err != nil {
		t.Errorf("installing jq asked: %v", err)
	}
}

func TestEveryConfirmationSaysWhatItsGestureDoes(t *testing.T) {
	for _, a := range confirm.Actions() {
		if c := a.Consequence(); c == "" || !strings.HasSuffix(c, ".") {
			t.Errorf("%s says %q: every confirmation says what it does, in one sentence", a, c)
		}
	}
	if got := confirm.RestartServer.Consequence(); got != "It is a server: what it serves stops while it restarts." {
		t.Errorf("a server's restart says %q: it says it is a server and what stops while it restarts", got)
	}
}
