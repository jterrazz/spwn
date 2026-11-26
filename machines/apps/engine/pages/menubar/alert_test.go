package menubar

import (
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

// value: the engine's rule as ADR-097 grades it, by gesture and facts,
// for a page test that holds no call.
func graded(gesture string, k machine.Kind, facts ...string) string {
	has := func(f string) bool { return slices.Contains(facts, f) }
	switch {
	case gesture == "delete" && has("ephemeral") && !has("running") && !has("held"):
		return LevelNone
	case gesture == "delete":
		return LevelTypeName
	case gesture == "stop" && k != machine.Server, gesture == "restart-server" && has("down"), gesture == "stop-sharing",
		gesture == "off-here" && !has(FactHostsServer):
		return LevelNone
	}
	return LevelYesNo
}

func consequence(c *page.Confirm) string {
	if c == nil || len(c.Lines) == 0 {
		return ""
	}
	return c.Lines[0].Text
}

func TestARunningEphemeralWorkspacesDeleteAsksItsNameTypedAndListsWhatGoesWithIt(t *testing.T) {
	vm := model.VM{Name: "agent-7f", Lifetime: model.Ephemeral, State: model.Running, SizeGB: 12}
	a := deleteWorkspace(vm, 2, 2).action(graded, page.Action{Run: &page.Run{In: map[string]string{"VM": "agent-7f"}}})
	if a.Confirm.Type != "agent-7f" || !a.Confirm.Danger || a.Confirm.Yes != "Delete agent-7f" {
		t.Fatalf("a running ephemeral workspace's delete asks %+v, want agent-7f typed, red, Delete agent-7f", a.Confirm)
	}
	if got := consequence(a.Confirm); got != "This can't be undone. It goes with its disk (12 GB), 2 shared ports and 2 grants made to it." {
		t.Errorf("an ephemeral workspace's delete says %q", got)
	}
	if _, fixed := a.Run.In["Confirm"]; fixed {
		t.Errorf("a typed delete fixes its answer %q: the owner's typed name is the call's", a.Run.In["Confirm"])
	}
}

func TestAStoppedEphemeralWorkspaceNobodyHoldsIsDeletedWithNoQuestion(t *testing.T) {
	vm := model.VM{Name: "agent-7f", Lifetime: model.Ephemeral, State: model.Stopped, SizeGB: 12}
	if a := deleteWorkspace(vm, 0, 0).action(graded, page.Action{Run: &page.Run{In: map[string]string{}}}); a.Confirm != nil || a.Run.In["Confirm"] != "" {
		t.Errorf("a stopped ephemeral workspace's delete asks %+v, sends %v", a.Confirm, a.Run.In)
	}
}
func TestAPersistentWorkspacesDeleteAsksItsNameTyped(t *testing.T) {
	vm := model.VM{Name: "api", Lifetime: model.Persistent, SizeGB: 24}
	a := deleteWorkspace(vm, 0, 0).action(graded, page.Action{Run: &page.Run{In: map[string]string{}}})
	if a.Confirm.Type != "api" || consequence(a.Confirm) != "This can't be undone. It goes with its disk (24 GB)." {
		t.Errorf("a persistent workspace's delete asks %+v", a.Confirm)
	}
	if _, fixed := a.Run.In["Confirm"]; fixed {
		t.Error("a typed delete fixes its answer: the owner's typed name is the call's")
	}
}

func TestDeletingAWorkspaceNamesWhatGoes(t *testing.T) {
	vm := model.VM{Name: "linux-1", Lifetime: model.Ephemeral, State: model.Running, SizeGB: 16}
	for _, c := range []struct {
		alert alert
		want  string
	}{
		{deleteWorkspace(vm, 2, 0), "This can't be undone. It goes with its disk (16 GB) and 2 shared ports."},
	} {
		if got := consequence(c.alert.confirm(graded)); got != c.want {
			t.Errorf("says %q, want %q", got, c.want)
		}
	}
}

func TestStopEverythingSaysWhatEndsOnThisMacAndWhatKeepsRunningOnAServersMac(t *testing.T) {
	this := stopEveryWorkspace("This Mac", true, []string{"linux-1", "agent-7f"}, 0).confirm(graded)
	mini := stopEveryWorkspace("mac-mini", false, []string{"linux-3", "review-bot"}, 1).confirm(graded)
	if consequence(this) != "linux-1, agent-7f stop and keep their disk. Awake ends." || this.Yes != "Stop 2 workspaces" {
		t.Errorf("This Mac's Stop everything asks %q, %q", consequence(this), this.Yes)
	}
	if consequence(mini) != "linux-3, review-bot stop and keep their disk. Servers keep running." {
		t.Errorf("mac-mini's Stop everything says %q", consequence(mini))
	}
}

func TestAnUpServersRestartAsksYesOrNoAndADownOneAsksNothing(t *testing.T) {
	up := model.VM{Name: "infra", Lifetime: model.AlwaysOn, State: model.Running}
	a := restartServer(up, "mac-mini").action(graded, page.Action{Run: &page.Run{In: map[string]string{"Name": "infra"}}})
	if a.Confirm == nil || a.Confirm.Type != "" || a.Confirm.Yes != "Restart infra" || a.Run.In["Confirm"] != yes {
		t.Errorf("an up server's restart asks %+v and sends %v, want yes or no, Restart infra, yes", a.Confirm, a.Run.In)
	}
	down := restartServer(up, "mac-mini")
	down.Facts = append(down.Facts, "down")
	if c := down.confirm(graded); c != nil {
		t.Errorf("a down server's restart asks %+v: a repair asks nothing", c)
	}
}

func TestAMacHostingAServerAsksYesOrNoNamingIt(t *testing.T) {
	a := restartMac("mac-mini", []string{"infra"}).action(graded, page.Action{Run: &page.Run{In: map[string]string{}}})
	if a.Confirm.Type != "" || consequence(a.Confirm) != "It hosts a server. infra is down while mac-mini restarts, about 2 min." {
		t.Errorf("mac-mini's restart asks %+v", a.Confirm)
	}
	if a.Run.In["Confirm"] != yes {
		t.Errorf("the restart call asks yes or no, so the alert sends yes: %q", a.Run.In["Confirm"])
	}
}
