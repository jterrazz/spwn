package wire

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	mffake "github.com/jterrazz/studio-station/apps/engine/machinefile/adapter/fake"
	machinefilecalls "github.com/jterrazz/studio-station/apps/engine/machinefile/calls"
)

func TestTheApplyGivenABoundItNeverReachesRunsWholeIsRecordedAndLetsTheMachineGo(t *testing.T) {
	// invariant: machinefile.apply itself, given a bound (none for zero or
	// less, as ApplyIn says) that its packages end inside, installs them,
	// records the run as one run to its end and lets the machine's run lock
	// go, so the next apply runs.
	for _, within := range []time.Duration{-time.Minute, 0, time.Hour} {
		b := benchOf(t, catalog.Mac, "jq")
		in := machinefilecalls.ApplyIn{Interactive: true, Within: within}
		applied, err := call.Invoke(context.Background(), b.calls, coremachine.Ref{}, machinefilecalls.Apply, in, nil)
		if err != nil {
			t.Fatalf("within %v: apply: %v", within, err)
		}
		i := slices.IndexFunc(applied.Done, func(d machinefile.Done) bool { return d.Line.Step.Package == "jq" })
		if i < 0 || applied.Done[i].State != catalog.StateOK || applied.Done[i].Reason != "" || applied.Interrupted {
			t.Fatalf("within %v: carried %+v; want jq installed, the run whole", within, applied.Done)
		}
		if _, ok, err := mffake.LedgerOf(b.w, "here").LastApply(context.Background(), "m"); err != nil || !ok {
			t.Fatalf("within %v: the run is not recorded as one run to its end (%v)", within, err)
		}
		if _, err := call.Invoke(context.Background(), b.calls, coremachine.Ref{}, machinefilecalls.Apply, in, nil); err != nil {
			t.Fatalf("within %v: the next apply: %v; want the machine let go", within, err)
		}
	}
}
