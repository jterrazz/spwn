package machinefile_test

import (
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/machinefile"
)

func placed(it catalog.Item) catalog.Placed { return catalog.Placed{Item: it} }

func TestJudgeTakesBackOnlyThisMachinefilesExtrasDependentsFirst(t *testing.T) {
	// invariant: an install of another Machinefile is no extra; what an
	// extra waits on (its manager, what it requires) is taken back after
	// it; only one there by the ledger's method is removed.
	cat := catalog.Catalog{Items: []catalog.Placed{
		placed(catalog.Item{Name: "manager", Manages: []catalog.ManagerKey{"pkg"}}),
		placed(catalog.Item{Name: "lib"}),
		placed(catalog.Item{Name: "tool", Requires: []string{"lib"}}),
	}}
	ledger := []machinefile.Installed{
		{Item: "lib", Method: "pkg", Machinefile: "laptop"},
		{Item: "manager", Method: "installer", Machinefile: "laptop"},
		{Item: "other", Method: "pkg", Machinefile: "old-laptop"},
		{Item: "tool", Method: "pkg", Machinefile: "laptop"},
	}
	seen := map[string]machinefile.Seen{
		"lib":     {Condition: machinefile.Present, Method: "pkg"},
		"manager": {Condition: machinefile.NotManaged, Method: "installer"},
		"tool":    {Condition: machinefile.Present, Method: "pkg"},
		"other":   {Condition: machinefile.Present, Method: "pkg"},
	}
	p := machinefile.Judge(machinefile.Resolved{Name: "laptop"}, cat, seen, ledger)
	order := make([]string, 0, 3)
	var removed []string
	for _, e := range p.Extra {
		order = append(order, e.Step.Item)
		if e.Takes() {
			removed = append(removed, e.Step.Item)
		}
	}
	if want := []string{"tool", "lib", "manager"}; !slices.Equal(order, want) {
		t.Errorf("extras in order %v; want %v", order, want)
	}
	if want := []string{"tool", "lib"}; !slices.Equal(removed, want) {
		t.Errorf("removed %v; want %v (the manager reads present another way)", removed, want)
	}
}

func TestJudgeKeepsAnItemSkippedOnThisSystem(t *testing.T) {
	// invariant: an optional item the Machinefile still lists is no extra
	// while this system skips it: no method here today is no removal.
	cat := catalog.Catalog{Items: []catalog.Placed{placed(catalog.Item{Name: "foo"})}}
	ledger := []machinefile.Installed{{Item: "foo", Method: "pkg", Machinefile: "laptop"}}
	seen := map[string]machinefile.Seen{"foo": {Condition: machinefile.Present, Method: "pkg"}}
	r := machinefile.Resolved{Name: "laptop", Skipped: []machinefile.Skip{{Item: "foo", Why: "optional, no method for macos 27"}}}
	for _, e := range machinefile.Judge(r, cat, seen, ledger).Extra {
		t.Errorf("%s is extra (%s); want none: it is still listed", e.Step.Item, e.Action)
	}
}

func TestJudgeKeepsAnExtraAnotherAppStillNeeds(t *testing.T) {
	// invariant: an extra an app there still needs (what requires it, what
	// its manager installed) is kept, unless that app is taken back too.
	cat := catalog.Catalog{Items: []catalog.Placed{
		placed(catalog.Item{Name: "manager", Kind: catalog.KindManager, Manages: []catalog.ManagerKey{"pkg"}}),
		placed(catalog.Item{Name: "lib"}),
		placed(catalog.Item{Name: "tool", Requires: []string{"lib"}}),
		placed(catalog.Item{Name: "own"}),
		placed(catalog.Item{Name: "dotfiles", Kind: catalog.KindSetting, Requires: []string{"own"}}),
	}}
	ledger := []machinefile.Installed{
		{Item: "manager", Method: "installer", Machinefile: "laptop"},
		{Item: "lib", Method: "pkg", Machinefile: "laptop"},
		{Item: "own", Method: "pkg", Machinefile: "laptop"},
	}
	seen := map[string]machinefile.Seen{
		"manager":  {Condition: machinefile.Present, Method: "installer"},
		"lib":      {Condition: machinefile.Present, Method: "pkg"},
		"tool":     {Condition: machinefile.Unknown, Method: "pkg"},
		"own":      {Condition: machinefile.Present, Method: "pkg"},
		"dotfiles": {Condition: machinefile.Present},
	}
	kept := map[string][]string{}
	for _, e := range machinefile.Judge(machinefile.Resolved{Name: "laptop"}, cat, seen, ledger).Extra {
		if !e.Takes() {
			kept[e.Step.Item] = e.RequiredBy
		}
	}
	want := map[string][]string{"lib": {"tool"}, "manager": {"lib", "tool"}}
	if len(kept) != len(want) || !slices.Equal(kept["lib"], want["lib"]) || !slices.Equal(kept["manager"], want["manager"]) {
		t.Errorf("kept %v; want %v (own is needed by a setting alone)", kept, want)
	}
}
