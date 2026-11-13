package settings_test

import (
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

func TestJudgeSetGivesEachSetByStationStateItsCause(t *testing.T) {
	starship := catalog.Setting{Name: "starship", Kinds: []catalog.MachineKind{catalog.Mac, catalog.Linux}}
	at := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	record := &settings.Applied{Setting: "starship", Digest: "sha256:station", At: at}
	same := settings.Observation{State: catalog.StateOK, Digest: "sha256:station", Where: "~/.config/starship.toml"}
	edited := settings.Observation{State: catalog.StateDegraded, Digest: "sha256:yours", Diff: "-a\n+b\n", Where: "~/.config/starship.toml"}
	gone := settings.Observation{State: catalog.StateAbsent, Where: "~/.config/starship.toml"}
	rewritten := settings.Observation{State: catalog.StateOK, Digest: "sha256:yours", Where: "~/.config/starship.toml"}
	earlier := settings.Observation{State: catalog.StateDegraded, Digest: "sha256:station", Diff: "-new\n+old\n", Where: "~/.config/starship.toml"}
	for _, c := range []struct {
		name    string
		lives   bool
		applied *settings.Applied
		now     settings.Observation
		off     bool
		want    settings.State
		since   bool
	}{
		{"applied and untouched since", true, record, same, false, settings.StateApplied, true},
		{"applied, then edited by hand", true, record, edited, false, settings.StateChangedSince, true},
		{"applied, then deleted by hand", true, record, gone, false, settings.StateChangedSince, true},
		{"applied, then rewritten by hand into a version that still reads in place", true, record, rewritten, false, settings.StateChangedSince, true},
		{"applied and untouched, station writing it otherwise since", true, record, earlier, false, settings.StateNotApplied, false},
		{"never applied, nothing there", true, nil, gone, false, settings.StateNotApplied, false},
		{"never applied, a file of your own there", true, nil, edited, false, settings.StateNotApplied, false},
		{"never recorded, but exactly what station writes", true, nil, same, false, settings.StateApplied, false},
		{"off here, even though applied", true, record, same, true, settings.StateOffThisMac, false},
		{"off here, your edit kept", true, record, edited, true, settings.StateOffThisMac, false},
		{"does not live on this machine, off or not", false, record, same, true, settings.StateNotForThisMachine, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			row := settings.JudgeSet(starship, c.lives, c.applied, c.now, c.off)
			if row.State != c.want {
				t.Fatalf("state = %s, want %s", row.State, c.want)
			}
			if c.since != !row.Since.IsZero() {
				t.Fatalf("since = %v, want it set: %v", row.Since, c.since)
			}
		})
	}
}

func TestJudgeSetReadsARunOnceSettingAsAppliedOnceStationRanIt(t *testing.T) {
	dock := catalog.Setting{Name: "dock-reset", Kinds: []catalog.MachineKind{catalog.Mac}, RunOnce: true}
	blind := settings.Observation{State: catalog.StateAbsent, Blind: true}
	if got := settings.JudgeSet(dock, true, nil, blind, false).State; got != settings.StateNotApplied {
		t.Fatalf("never run: %s, want not-applied", got)
	}
	ran := &settings.Applied{Setting: "dock-reset", At: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)}
	if got := settings.JudgeSet(dock, true, ran, blind, false).State; got != settings.StateApplied {
		t.Fatalf("ran once: %s, want applied", got)
	}
}

func TestJudgeSetNamesWhereASettingLivesWhenItIsNotForThisMachine(t *testing.T) {
	for places, want := range map[string]struct {
		kinds []catalog.MachineKind
		why   string
	}{
		"a Mac's only":  {[]catalog.MachineKind{catalog.Mac}, "Mac only"},
		"Macs and VMs":  {[]catalog.MachineKind{catalog.Mac, catalog.MacOS}, "Macs and macOS VMs only"},
		"the VMs' only": {[]catalog.MachineKind{catalog.Linux}, "VMs only"},
	} {
		row := settings.JudgeSet(catalog.Setting{Name: "x", Kinds: want.kinds}, false, nil, settings.Observation{}, false)
		if row.Why != want.why {
			t.Errorf("%s: why = %q, want %q", places, row.Why, want.why)
		}
	}
}

func TestARowFromAnOlderStationIsShownWithThisStationsItem(t *testing.T) {
	local := func(name string) (catalog.Setting, bool) {
		if name == "hushlogin" {
			return catalog.Setting{Name: "hushlogin", Description: "Silence the terminal login message"}, true
		}
		return catalog.Setting{}, false
	}
	far := []settings.Row{
		{Name: "hushlogin", State: settings.StateApplied},
		{Name: "dns", Item: catalog.Setting{Name: "dns", Description: "its own words"}},
		{Name: "gone", State: settings.StateApplied},
	}
	got := settings.Known(far, local)
	if got[0].Item.Description != "Silence the terminal login message" || got[1].Item.Description != "its own words" || got[2].Item.Name != "" {
		t.Errorf("rows filled as %+v", got)
	}
	if far[0].Item.Name != "" {
		t.Error("the rows given were changed")
	}
}
