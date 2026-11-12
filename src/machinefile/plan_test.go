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
	p := machinefile.Judge(machinefile.Resolved{Name: "laptop"}, cat, seen, ledger, false)
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
	for _, e := range machinefile.Judge(r, cat, seen, ledger, false).Extra {
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
	for _, e := range machinefile.Judge(machinefile.Resolved{Name: "laptop"}, cat, seen, ledger, false).Extra {
		if !e.Takes() {
			kept[e.Step.Item] = e.RequiredBy
		}
	}
	want := map[string][]string{"lib": {"tool"}, "manager": {"lib", "tool"}}
	if len(kept) != len(want) || !slices.Equal(kept["lib"], want["lib"]) || !slices.Equal(kept["manager"], want["manager"]) {
		t.Errorf("kept %v; want %v (own is needed by a setting alone)", kept, want)
	}
}

func TestJudgeKeepsTheProviderOfAListedApp(t *testing.T) {
	// invariant: an app provided by an extra leans on it like on what it
	// requires: the provider stays while that app is there.
	cat := catalog.Catalog{Items: []catalog.Placed{
		placed(catalog.Item{Name: "manager", Kind: catalog.KindManager, Manages: []catalog.ManagerKey{"pkg"}}),
		placed(catalog.Item{Name: "desktop"}),
		placed(catalog.Item{Name: "core", Methods: []catalog.MethodSpec{{ProvidedBy: "desktop"}}}),
	}}
	ledger := []machinefile.Installed{{Item: "desktop", Method: "pkg", Machinefile: "laptop"}}
	seen := map[string]machinefile.Seen{
		"manager": {Condition: machinefile.Present, Method: "script"},
		"desktop": {Condition: machinefile.Present, Method: "pkg"},
		"core":    {Condition: machinefile.Present, Method: "provided-by"},
	}
	r := machinefile.Resolved{Name: "laptop", Steps: []machinefile.Step{{Item: "core", Kind: catalog.KindApp, Method: "provided-by", Phase: catalog.PhaseApply}}}
	for _, e := range machinefile.Judge(r, cat, seen, ledger, false).Extra {
		if e.Takes() || !slices.Equal(e.RequiredBy, []string{"core"}) {
			t.Errorf("%s: takes %v, required by %v; want kept, required by [core]", e.Step.Item, e.Takes(), e.RequiredBy)
		}
	}
}

func TestJudgeKeepsWhatAnExtraStationCannotUninstallNeeds(t *testing.T) {
	// invariant: an extra station cannot uninstall stays, so what it
	// requires is never taken back under it.
	cat := catalog.Catalog{Items: []catalog.Placed{
		placed(catalog.Item{Name: "runtime"}),
		placed(catalog.Item{Name: "browsers", Requires: []string{"runtime"}}),
	}}
	ledger := []machinefile.Installed{
		{Item: "runtime", Method: "pkg", Machinefile: "laptop"},
		{Item: "browsers", Method: "script", Machinefile: "laptop"},
	}
	seen := map[string]machinefile.Seen{
		"runtime":  {Condition: machinefile.Present, Method: "pkg"},
		"browsers": {Condition: machinefile.Present, Method: "script", Irreversible: true},
	}
	fates := map[string]machinefile.Fate{}
	for _, e := range machinefile.Judge(machinefile.Resolved{Name: "laptop"}, cat, seen, ledger, false).Extra {
		fates[e.Step.Item] = e.Fate()
		if e.Step.Item == "runtime" && !slices.Equal(e.RequiredBy, []string{"browsers"}) {
			t.Errorf("runtime required by %v; want [browsers]", e.RequiredBy)
		}
	}
	if fates["runtime"] != machinefile.KeptRequired || fates["browsers"] != machinefile.KeptIrreversible {
		t.Errorf("fates %v; want runtime kept-required, browsers kept-irreversible", fates)
	}
}

func TestJudgeGivesEveryExtraOneFateWhetherAnAppNeedsIt(t *testing.T) {
	// invariant: every way an extra reads is judged at plan time, with an
	// app there that needs it and without: one is taken back alone, asking
	// what its removal asks; what needs one that stays there by station's
	// method is said and keeps it, the one station cannot uninstall
	// included; nothing else is ever asked, nor removed but its record.
	cat := catalog.Catalog{Items: []catalog.Placed{
		placed(catalog.Item{Name: "manager", Kind: catalog.KindManager, Manages: []catalog.ManagerKey{"pkg"}}),
		placed(catalog.Item{Name: "lib", Needs: []catalog.Need{catalog.NeedSudo}}),
		placed(catalog.Item{Name: "tool", Requires: []string{"lib"}}),
	}}
	ledger := []machinefile.Installed{{Item: "lib", Method: "pkg", Machinefile: "laptop"}}
	for _, c := range []struct {
		reads           string
		lib             machinefile.Seen
		alone, neededBy machinefile.Fate
	}{
		{"there by its method", machinefile.Seen{Condition: machinefile.Present, Method: "pkg"}, machinefile.Taken, machinefile.KeptRequired},
		{"there, station cannot uninstall it", machinefile.Seen{Condition: machinefile.Present, Method: "pkg", Irreversible: true}, machinefile.KeptIrreversible, machinefile.KeptRequired},
		{"absent, its manager missing", machinefile.Seen{Condition: machinefile.Absent, Method: "pkg", Lacks: "manager"}, machinefile.KeptManagerMissing, machinefile.KeptManagerMissing},
		{"already gone", machinefile.Seen{Condition: machinefile.Absent, Method: "pkg"}, machinefile.Gone, machinefile.Gone},
		{"there another way", machinefile.Seen{Condition: machinefile.NotManaged, Method: "script"}, machinefile.KeptAnotherWay, machinefile.KeptAnotherWay},
		{"there, a manager holding a package of yours", machinefile.Seen{Condition: machinefile.Present, Method: "pkg", Holds: []string{"typescript"}}, machinefile.KeptHolding, machinefile.KeptRequired},
		{"there, a manager whose packages do not read", machinefile.Seen{Condition: machinefile.Present, Method: "pkg", HoldsUnread: true}, machinefile.KeptHolding, machinefile.KeptRequired},
		{"unread", machinefile.Seen{Condition: machinefile.Unknown, Method: "pkg"}, machinefile.KeptUnread, machinefile.KeptUnread},
	} {
		for _, needed := range []bool{false, true} {
			tool, want := machinefile.Seen{Condition: machinefile.Absent}, c.alone
			if needed {
				tool, want = machinefile.Seen{Condition: machinefile.Present, Method: "pkg"}, c.neededBy
			}
			seen := map[string]machinefile.Seen{"manager": {Condition: machinefile.Present, Method: "script"}, "lib": c.lib, "tool": tool}
			p := machinefile.Judge(machinefile.Resolved{Name: "laptop"}, cat, seen, ledger, false)
			e := p.Extra[0]
			if got := e.Fate(); got != want {
				t.Errorf("lib %s, needed %v: %s; want %s", c.reads, needed, got, want)
			}
			if by := []string{"tool"}; (want == machinefile.KeptRequired) != slices.Equal(e.RequiredBy, by) {
				t.Errorf("lib %s, needed %v: required by %v; want %v only when kept for it", c.reads, needed, e.RequiredBy, by)
			}
			asked := len(p.Interactive(true)) > 0
			if asked != (want == machinefile.Taken) || len(p.Needs) > 0 {
				t.Errorf("lib %s, needed %v: --remove asks %v, plan asks %v; want asked only when taken back", c.reads, needed, p.Interactive(true), p.Needs)
			}
			if removes := e.Action == machinefile.Remove; removes != (want == machinefile.Taken || want == machinefile.Gone) {
				t.Errorf("lib %s, needed %v: %s; want remove only when taken back or gone", c.reads, needed, e.Action)
			}
			if forgets := slices.Contains([]machinefile.Fate{machinefile.Taken, machinefile.Gone, machinefile.KeptIrreversible, machinefile.KeptAnotherWay}, want); p.Calm() == forgets {
				t.Errorf("lib %s, needed %v: calm %v; want calm only when --remove changes nothing", c.reads, needed, p.Calm())
			}
		}
	}
}

func TestJudgeAsksThePasswordOnlyWhereTheMachineAsksOne(t *testing.T) {
	// invariant: a sudo need asks the owner only where the machine said a
	// line run as root asks a password; every other need still asks.
	cat := catalog.Catalog{Items: []catalog.Placed{
		placed(catalog.Item{Name: "manager", Needs: []catalog.Need{catalog.NeedSudo}}),
		placed(catalog.Item{Name: "editor", Needs: []catalog.Need{catalog.NeedLicense}}),
		placed(catalog.Item{Name: "old", Needs: []catalog.Need{catalog.NeedSudo}}),
	}}
	r := machinefile.Resolved{Name: "laptop", Steps: []machinefile.Step{
		{Item: "manager", Kind: catalog.KindApp, Method: "script", Phase: catalog.PhaseApply, Needs: []catalog.Need{catalog.NeedSudo}},
		{Item: "editor", Kind: catalog.KindApp, Method: "pkg", Phase: catalog.PhaseApply, Needs: []catalog.Need{catalog.NeedLicense}},
	}}
	seen := map[string]machinefile.Seen{
		"manager": {Condition: machinefile.Absent},
		"editor":  {Condition: machinefile.Absent},
		"old":     {Condition: machinefile.Present, Method: "pkg"},
	}
	ledger := []machinefile.Installed{{Item: "old", Method: "pkg", Machinefile: "laptop"}}
	needs := func(a []machinefile.Asked) []catalog.Need {
		out := make([]catalog.Need, 0, len(a))
		for _, n := range a {
			out = append(out, n.Need)
		}
		return out
	}
	for _, c := range []struct {
		passwordless bool
		plan, remove []catalog.Need
	}{
		{false, []catalog.Need{catalog.NeedSudo, catalog.NeedLicense}, []catalog.Need{catalog.NeedSudo, catalog.NeedLicense}},
		{true, []catalog.Need{catalog.NeedLicense}, []catalog.Need{catalog.NeedLicense}},
	} {
		p := machinefile.Judge(r, cat, seen, ledger, c.passwordless)
		if got := needs(p.Needs); !slices.Equal(got, c.plan) {
			t.Errorf("passwordless %v: the plan asks %v; want %v", c.passwordless, got, c.plan)
		}
		if got := needs(p.Interactive(true)); !slices.Equal(got, c.remove) {
			t.Errorf("passwordless %v: --remove asks %v; want %v", c.passwordless, got, c.remove)
		}
		if p.Asks(catalog.NeedSudo) == c.passwordless || !p.Asks(catalog.NeedLicense) {
			t.Errorf("passwordless %v: asks sudo %v, a license %v", c.passwordless, p.Asks(catalog.NeedSudo), p.Asks(catalog.NeedLicense))
		}
	}
}

func TestUpgradeAsksOnlyWhatTheUpdateNeedsNotItsInstalls(t *testing.T) {
	// invariant: an upgrade asks the item's needs only for an app it
	// installs; an app it updates asks its method's needs alone.
	cat := catalog.Catalog{Items: []catalog.Placed{
		placed(catalog.Item{Name: "manager", Needs: []catalog.Need{catalog.NeedSudo}}),
		placed(catalog.Item{Name: "gone", Needs: []catalog.Need{catalog.NeedSudo}}),
		placed(catalog.Item{Name: "pkg"}),
	}}
	step := func(item string, spec catalog.MethodSpec, needs ...catalog.Need) machinefile.Step {
		return machinefile.Step{Item: item, Kind: catalog.KindApp, Method: "script", Spec: spec, Phase: catalog.PhaseApply, Lock: "station.lock", Needs: needs}
	}
	r := machinefile.Resolved{Name: "laptop", Steps: []machinefile.Step{
		step("manager", catalog.MethodSpec{}, catalog.NeedSudo),
		step("gone", catalog.MethodSpec{}, catalog.NeedSudo),
		step("pkg", catalog.MethodSpec{Needs: []catalog.Need{catalog.NeedLicense}}, catalog.NeedLicense),
	}}
	seen := map[string]machinefile.Seen{
		"manager": {Condition: machinefile.Present, Method: "script"},
		"gone":    {Condition: machinefile.Absent},
		"pkg":     {Condition: machinefile.Present, Method: "script"},
	}
	p := machinefile.Judge(r, cat, seen, nil, false)
	got := p.UpgradeNeeds()
	if len(got) != 2 || got[0].Need != catalog.NeedSudo || !slices.Equal(got[0].Items, []string{"gone"}) || got[1].Need != catalog.NeedLicense || !slices.Equal(got[1].Items, []string{"pkg"}) {
		t.Errorf("upgrade asks %+v; want sudo for gone (installed) and a license for pkg (its method's)", got)
	}
}
