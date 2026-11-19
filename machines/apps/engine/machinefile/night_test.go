package machinefile_test

import (
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

func TestANightThatDidNotEndKeepsTheLastOneThatDid(t *testing.T) {
	// invariant: a failed or interrupted night keeps the last complete
	// night of its Machinefile beside it, one past another failure too;
	// a complete night, or one of another Machinefile, keeps none.
	good := machinefile.Night{At: time.Unix(1, 0), Machinefile: "m", Moved: []machinefile.Moved{{App: "tool"}}, Waiting: []machinefile.Waiting{{App: "node"}}}
	failed := machinefile.Night{Machinefile: "m", Error: "does not resolve"}
	first := failed.After(good, true)
	if first.Last == nil || first.Last.Waiting[0].App != "node" || first.Last.Moved[0].App != "tool" {
		t.Fatalf("a failed night dropped the last good one: %+v", first.Last)
	}
	busy := machinefile.Night{Machinefile: "m", Error: "busy"}
	if again := busy.After(first, true); again.Last == nil || again.Last.At != good.At {
		t.Errorf("a second failure dropped the last good night: %+v", again.Last)
	}
	if interrupted := (machinefile.Night{Machinefile: "m", Interrupted: true}).After(good, true); interrupted.Last == nil {
		t.Error("an interrupted night dropped the last good one")
	}
	if done := (machinefile.Night{Machinefile: "m"}).After(first, true); done.Last != nil {
		t.Errorf("a complete night kept one beside it: %+v", done.Last)
	}
	if other := (machinefile.Night{Machinefile: "other", Error: "x"}).After(good, true); other.Last != nil {
		t.Error("a night of another Machinefile kept this one's")
	}
	if none := failed.After(machinefile.Night{}, false); none.Last != nil {
		t.Error("a first night kept one it never had")
	}
	if refreshed := first.Refreshed(machinefile.Upgraded{}); refreshed.Last != nil {
		t.Error("an owner's upgrade run to its end kept the last night beside it")
	}
}

func TestAnApplyReadsWhatWaitsAgainstTheRangesItApplied(t *testing.T) {
	// invariant: a version past the range its app's line holds now still
	// waits, in that range's words; one the line's range was moved to
	// hold, or of an app listed no longer, waits no more.
	line := func(pkg, version string) machinefile.Line {
		r, err := catalog.ParseRange(version)
		if err != nil {
			t.Fatal(err)
		}
		var l machinefile.Line
		l.Step.Package, l.Step.Version = pkg, r
		return l
	}
	n := machinefile.Night{Machinefile: "m", Waiting: []machinefile.Waiting{
		{App: "node", Newest: "25.1.0", Range: "24", Moved: "25"},
		{App: "go", Newest: "1.26.0", Range: "~1.25.0", Moved: "~1.26.0"},
		{App: "gone", Newest: "2.0.0", Range: "1", Moved: "2"},
	}}
	n.Last = &machinefile.Night{Machinefile: "m", Waiting: n.Waiting}
	got := n.Reread(machinefile.Plan{Lines: []machinefile.Line{line("node", "25"), line("go", "1.25")}})
	want := machinefile.Waiting{App: "go", Newest: "1.26.0", Range: "1.25", Moved: "1.26"}
	if len(got.Waiting) != 1 || got.Waiting[0] != want {
		t.Errorf("waits %+v; want only %+v", got.Waiting, want)
	}
	if got.Last == nil || len(got.Last.Waiting) != 1 {
		t.Errorf("the last night kept beside was not read again: %+v", got.Last)
	}
}
