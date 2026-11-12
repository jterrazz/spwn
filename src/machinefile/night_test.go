package machinefile_test

import (
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/machinefile"
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
