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
