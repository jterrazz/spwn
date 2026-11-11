package machinefile_test

import (
	"reflect"
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/machinefile"
)

func TestARunThatStopsKeepsTheLockOfWhatItDidNotReach(t *testing.T) {
	// invariant: on every way a line ends, the lock takes what reads
	// there by a method and keeps the rest as it held it: an install that
	// finished (before an interrupt too), one interrupted, failed or not
	// run, a line kept, a copy not managed, an app listed no longer.
	old := machinefile.Lock{LockFile: catalog.LockFile{Machinefile: "m", Apps: []catalog.LockedApp{
		{Name: "done", Method: "pkg", Version: "1.0"}, {Name: "stopped", Method: "pkg", Version: "1.0"},
		{Name: "failed", Method: "pkg", Version: "1.0"}, {Name: "skipped", Method: "pkg", Version: "1.0"},
		{Name: "yours", Method: "pkg", Version: "1.0"}, {Name: "gone", Method: "pkg", Version: "1.0"},
	}}}
	line := func(item string, a machinefile.Action, seen machinefile.Seen) machinefile.Line {
		return machinefile.Line{Step: machinefile.Step{Item: item, Kind: catalog.KindApp, Method: "pkg", Phase: catalog.PhaseApply}, Action: a, Seen: seen}
	}
	absent := machinefile.Seen{Condition: machinefile.Absent, State: catalog.StateAbsent}
	p := machinefile.Plan{Name: "m", Lock: old, Lines: []machinefile.Line{
		line("done", machinefile.Install, absent),
		line("stopped", machinefile.Install, absent),
		line("failed", machinefile.Install, absent),
		line("skipped", machinefile.Install, absent),
		line("kept", machinefile.Keep, machinefile.Seen{Condition: machinefile.Present, State: catalog.StateOK, Method: "box", Version: "3.0"}),
		line("yours", machinefile.Adopt, machinefile.Seen{Condition: machinefile.NotManaged, State: catalog.StateOK, Method: "pkg"}),
		line("new", machinefile.Install, absent),
	}}
	done := []machinefile.Done{
		{Line: p.Lines[0], State: catalog.StateOK, Method: "pkg", Version: "2.0", Installed: true},
		{Line: p.Lines[1], State: catalog.StateBroken, Reason: machinefile.Interrupted},
		{Line: p.Lines[2], State: catalog.StateBroken, Detail: "exited 1"},
		{Line: p.Lines[3], State: catalog.StateAbsent, Reason: machinefile.NotRun},
		{Line: p.Lines[4], State: catalog.StateOK, Version: "3.0"},
		{Line: p.Lines[5], State: catalog.StateOK},
		{Line: p.Lines[6], State: catalog.StateOK, Method: "pkg", Version: "0.1", Installed: true},
	}
	got := machinefile.Relocked(p, done, nil)
	want := []catalog.LockedApp{
		{Name: "done", Method: "pkg", Version: "2.0"}, {Name: "failed", Method: "pkg", Version: "1.0"},
		{Name: "kept", Method: "box", Version: "3.0"}, {Name: "new", Method: "pkg", Version: "0.1"},
		{Name: "skipped", Method: "pkg", Version: "1.0"}, {Name: "stopped", Method: "pkg", Version: "1.0"},
	}
	if !reflect.DeepEqual(got.Apps, want) {
		t.Errorf("Relocked = %+v\nwant       %+v", got.Apps, want)
	}
	d := machinefile.Diff(old.LockFile, got)
	moved := make([]string, 0, len(d.Apps))
	for _, c := range d.Apps {
		moved = append(moved, c.App)
	}
	if !reflect.DeepEqual(moved, []string{"done", "gone", "kept", "new", "yours"}) {
		t.Errorf("Diff moved %v", moved)
	}
}

func TestALockedVersionMeetsTheOneAskedOnlyUnderIt(t *testing.T) {
	for asked, cases := range map[string]map[string]bool{
		"":      {"1.0": true},
		"22":    {"22": true, "22.11.0": true, "220.1": false, "2": false},
		"1.4.4": {"1.4.4": true, "1.4.40": false},
	} {
		for locked, want := range cases {
			if got := machinefile.Satisfies(asked, locked); got != want {
				t.Errorf("Satisfies(%q, %q) = %v", asked, locked, got)
			}
		}
	}
}
