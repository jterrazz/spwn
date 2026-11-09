package menubar

import (
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/src/core/machine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
)

// value: the engine's rule as SPEC A5 grades it, by gesture and facts; with
// no fact, the call's own level.
func graded(gesture string, k machine.Kind, facts ...string) string {
	has := func(f string) bool { return slices.Contains(facts, f) }
	switch {
	case gesture == "delete" && k == machine.Workspace && has(FactThrowaway):
		return LevelYesNo
	case gesture == "delete", gesture == "stop" && k == machine.Server:
		return LevelTypeName
	case gesture == "restart" && k == machine.Server:
		if has(FactHealthy) {
			return LevelYesNo
		}
		return LevelNone
	case gesture == "restart-mac" && has(FactHostsServer):
		return LevelTypeName
	}
	return LevelYesNo
}

func consequence(c *page.Confirm) string {
	if c == nil || len(c.Lines) == 0 {
		return ""
	}
	return c.Lines[0].Text
}

func TestAThrowawaysDeleteAsksYesOrNoAndListsWhatGoesWithIt(t *testing.T) {
	vm := model.VM{Name: "agent-7f", Lifetime: model.Throwaway, SizeGB: 12}
	a := deleteWorkspace(vm, nil, 2, 2).action(graded, page.Action{Run: &page.Run{In: map[string]string{"VM": "agent-7f"}}})
	if a.Confirm.Type != "" || !a.Confirm.Danger || a.Confirm.Yes != "Delete agent-7f" {
		t.Fatalf("a throwaway's delete asks %+v, want yes or no, red, Delete agent-7f", a.Confirm)
	}
	if got := consequence(a.Confirm); got != "This can't be undone. It goes with its disk (12 GB), 2 shared ports and 2 accounts lent to it." {
		t.Errorf("a throwaway's delete says %q", got)
	}
	if a.Run.In["Confirm"] != `"agent-7f"` {
		t.Errorf("the delete call asks the name typed, so yes sends it: %q", a.Run.In["Confirm"])
	}
}

func TestAPermanentWorkspacesDeleteAsksItsNameTyped(t *testing.T) {
	vm := model.VM{Name: "api", Lifetime: model.Permanent, SizeGB: 24}
	a := deleteWorkspace(vm, nil, 0, 0).action(graded, page.Action{Run: &page.Run{In: map[string]string{}}})
	if a.Confirm.Type != "api" || consequence(a.Confirm) != "This can't be undone. It goes with its disk (24 GB)." {
		t.Errorf("a permanent workspace's delete asks %+v", a.Confirm)
	}
	if _, fixed := a.Run.In["Confirm"]; fixed {
		t.Error("a typed delete fixes its answer: the owner's typed name is the call's")
	}
}

func TestStoppingOrDeletingAVMNamesTheAgentsThatStopWithIt(t *testing.T) {
	vm := model.VM{Name: "linux-1", Lifetime: model.Throwaway, SizeGB: 16}
	for _, c := range []struct {
		alert alert
		want  string
	}{
		{deleteWorkspace(vm, []string{"linux-1", "docs-bot"}, 2, 0), "Its agents stop first: linux-1, docs-bot. This can't be undone. It goes with its disk (16 GB) and 2 shared ports."},
		{stopWorkspace(vm, "This Mac", []string{"linux-1"}, 2), "Its agent linux-1 stops. Its disk is kept. 2 shared ports stop answering until it starts again."},
		{stopWorkspace(vm, "This Mac", nil, 1), "Its disk is kept. 1 shared port stops answering until it starts again."},
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

func TestAServersRestartAsksOnlyWhileItIsHealthy(t *testing.T) {
	vm := model.VM{Name: "infra", Lifetime: model.AlwaysOn}
	if c := restartServer(vm, "mac-mini", true).confirm(graded); c == nil || c.Danger || c.Yes != "Restart infra" {
		t.Errorf("a healthy server's restart asks %+v, want yes or no with Restart infra tinted", c)
	}
	if c := restartServer(vm, "mac-mini", false).confirm(graded); c != nil {
		t.Errorf("a down server's restart asks %+v, want none", c)
	}
}

func TestAMacHostingAServerAsksItsNameTypedYetItsCallTakesYes(t *testing.T) {
	a := restartMac("mac-mini", []string{"infra"}).action(graded, page.Action{Run: &page.Run{In: map[string]string{}}})
	if a.Confirm.Type != "mac-mini" || consequence(a.Confirm) != "It hosts a server. infra is down while mac-mini restarts, about 2 min." {
		t.Errorf("mac-mini's restart asks %+v", a.Confirm)
	}
	if a.Run.In["Confirm"] != yes {
		t.Errorf("the restart call asks yes or no, so the alert sends yes: %q", a.Run.In["Confirm"])
	}
	if c := restartMac("macbook-air", nil).confirm(graded); c.Type != "" {
		t.Errorf("a Mac hosting no server asks %+v, want yes or no", c)
	}
}
