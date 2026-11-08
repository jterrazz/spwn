package menubar

import (
	"slices"
	"strings"
	"testing"

	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/overview"
	"github.com/jterrazz/studio-station/src/page"
)

// invariant: EN-10: Stop everything names This Mac, whatever its name, and
// says a server running there is skipped and keeps the engine running; a
// server on another Mac is not its business.
func TestStopEverythingOnThisMacSaysItsServerIsSkipped(t *testing.T) {
	fleet := &model.AllMachines{Macs: []model.MacView{
		{Name: "jterrazz-macbook-pro", ThisMac: true, VMs: []model.VM{
			{Name: "infra", Lifetime: model.AlwaysOn, State: model.Running},
			{Name: "linux-1", Lifetime: model.Throwaway, State: model.Running},
		}},
		{Name: "jterrazz-mac-mini", VMs: []model.VM{{Name: "db", Lifetime: model.AlwaysOn, State: model.Running}}},
	}}
	confirm := stopAll(Reading{All: fleet}).Confirm
	if confirm.Question != "Stop everything on This Mac?" {
		t.Fatalf("question %q", confirm.Question)
	}
	said := make([]string, 0, len(confirm.Lines))
	for _, l := range confirm.Lines {
		said = append(said, l.Text)
	}
	want := []string{"linux-1 will stop; its disk is kept", "infra skipped (server)", "Awake ends, and the engine keeps running for infra"}
	if !slices.Equal(said, want) {
		t.Fatalf("lines %q, want %q", said, want)
	}
}

// invariant: AP-23: once Update all has run, the line of the app updates
// leaves Needs you and Update all goes with it; the badge, which never
// counts upkeep, says nothing of either.
func TestUpdateAllLeavesNeedsYouAndTheBadgeNeverCountedIt(t *testing.T) {
	needs := []overview.Need{
		{Section: "apps", Subject: "updates", Text: "2 app updates", Call: CallUpdate},
		{Section: "disk", Subject: "misplaced", Text: "a folder is misplaced", Call: "disk.move"},
	}
	before, after := overview.Summary{Needs: needs}, overview.Summary{Needs: needs[1:]}
	fleet := &model.AllMachines{Macs: []model.MacView{{Name: "jterrazz-macbook-pro", ThisMac: true}}}
	was, is := Page(Reading{Home: &before, All: fleet}, page.Context{}), Page(Reading{Home: &after, All: fleet}, page.Context{})
	if was.Badge.Needs != 0 || is.Badge.Needs != 0 {
		t.Fatalf("badge %d then %d, want 0 then 0", was.Badge.Needs, is.Badge.Needs)
	}
	has := func(p page.Page) bool {
		return slices.ContainsFunc(p.Actions, func(a page.Action) bool { return strings.HasPrefix(a.ID, ActionUpdateAll) && a.Run.Call == CallUpdate })
	}
	if !has(was) || has(is) {
		t.Fatalf("Update all before %v, after %v", has(was), has(is))
	}
}

// invariant: MA-14: Home draws each Mac's servers, then its workspaces,
// whatever order its engine lists them in.
func TestHomeDrawsAMacsServersThenItsWorkspaces(t *testing.T) {
	fleet := &model.AllMachines{Macs: []model.MacView{
		{Name: "jterrazz-mac-mini", ThisMac: true, Reachable: true, VMs: []model.VM{
			{Name: "build", Lifetime: model.Throwaway, State: model.Running},
			{Name: "infra", Lifetime: model.AlwaysOn, State: model.Running},
		}},
	}}
	p := Page(Reading{All: fleet}, page.Context{})
	var drawn []string
	for _, s := range p.Sections {
		if s.ID != SectionMachines {
			continue
		}
		for _, mac := range s.Items {
			for _, g := range mac.Group.Items {
				names := []string{g.Group.Head.Label}
				for _, vm := range g.Group.Items {
					names = append(names, vm.Row.Label)
				}
				drawn = append(drawn, names...)
			}
		}
	}
	if want := []string{GroupServers, "infra", GroupSpaces, "build"}; !slices.Equal(drawn, want) {
		t.Fatalf("Home draws %q, want %q", drawn, want)
	}
}

// invariant: MB-16: each update line's Update all updates what it counts,
// its manager's updates, never the other line's too.
func TestEachUpdateLineUpdatesItsOwnManager(t *testing.T) {
	needs := []overview.Need{
		{Section: "apps", Subject: "tools", Text: "1 tools update waiting", Call: CallUpdate, Args: map[string]string{"method": "tools"}},
		{Section: "apps", Subject: "pkgs", Text: "2 pkgs updates waiting", Call: CallUpdate, Args: map[string]string{"method": "pkgs"}},
	}
	p := Page(Reading{Home: &overview.Summary{Needs: needs}}, page.Context{})
	got := map[string]string{}
	for _, a := range p.Actions {
		if strings.HasPrefix(a.ID, ActionUpdateAll) {
			got[strings.Join(a.Rows, ",")] = a.Run.In["Method"]
		}
	}
	if len(got) != 2 || got["0-tools"] != `["tools"]` || got["1-pkgs"] != `["pkgs"]` {
		t.Fatalf("Update all per line: %v, want tools's on its line and pkgs's on its own", got)
	}
}
