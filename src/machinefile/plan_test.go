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
