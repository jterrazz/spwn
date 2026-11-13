package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

type nights struct {
	clock   *clock
	bound   string
	since   time.Time
	last    *machinefile.Night
	ran     []string
	records bool
	cancel  bool
	failure string
	unread  error
	reads   int
	held    []func()
}

func (n *nights) job(started time.Time) *Nightly {
	return &Nightly{
		Clock: n.clock, Started: started,
		Bound: func() (string, time.Time, bool) { return n.bound, n.since, n.bound != "" },
		Last: func(context.Context) (machinefile.Night, bool, error) {
			n.reads++
			if n.unread != nil {
				return machinefile.Night{}, false, n.unread
			}
			if n.last == nil {
				return machinefile.Night{}, false, nil
			}
			return *n.last, true, nil
		},
		Upgrade: func(_ context.Context, name string) error {
			n.ran = append(n.ran, name)
			if n.records {
				n.last = &machinefile.Night{At: n.clock.Now(), Machinefile: name, Interrupted: n.cancel, Error: n.failure}
			}
			return nil
		},
		Go: func(f func()) { n.held = append(n.held, f) },
	}
}

func (n *nights) finish() {
	for _, f := range n.held {
		f()
	}
	n.held = nil
}

func at(day, hour, minute int) time.Time {
	return time.Date(2026, 11, day, hour, minute, 0, 0, time.UTC)
}

func TestTheNightUpgradesTheBoundMachinefileOncePastThreeAndOnceAtATime(t *testing.T) {
	// invariant: the night is due at the first 03:00 past the engine's
	// start and the binding, or past the last night that completed; one runs
	// at a time; a Mac asleep at 03:00 runs it when it is back; a machine
	// bound to no Machinefile has none.
	n := &nights{clock: &clock{now: at(11, 23, 0)}, records: true}
	job := n.job(at(9, 10, 0))
	job.Run(context.Background())
	if len(n.held) != 0 {
		t.Fatalf("a machine bound to none started a night")
	}
	n.bound, n.since = "laptop", at(11, 22, 0)
	job.Run(context.Background())
	if len(n.held) != 0 {
		t.Fatalf("a night started at 23:00, an hour after the machine was bound")
	}
	n.clock.now = at(12, 9, 30)
	job.Run(context.Background())
	job.Run(context.Background())
	if len(n.held) != 1 {
		t.Fatalf("started %d nights past 03:00; want one, and none while it runs", len(n.held))
	}
	n.finish()
	n.clock.now = at(12, 22, 0)
	job.Run(context.Background())
	if len(n.held) != 0 || len(n.ran) != 1 {
		t.Fatalf("ran %v by 22:00 after the night of 09:30; want it once", n.ran)
	}
	n.clock.now = at(13, 3, 0)
	job.Run(context.Background())
	n.finish()
	if len(n.ran) != 2 {
		t.Errorf("ran %v at the next 03:00; want a second night", n.ran)
	}
}

func TestAnInterruptedNightRunsAgainAtTheNextStart(t *testing.T) {
	// invariant: a night the engine's stop interrupted did not complete:
	// the next engine to start runs it again, though 03:00 is behind it.
	n := &nights{clock: &clock{now: at(12, 3, 0)}, bound: "laptop", since: at(11, 10, 0), records: true, cancel: true}
	n.job(at(11, 10, 0)).Run(context.Background())
	n.finish()
	n.cancel = false
	n.clock.now = at(12, 7, 0)
	n.job(at(12, 7, 0)).Run(context.Background())
	n.finish()
	if len(n.ran) != 2 || n.last.Interrupted {
		t.Errorf("ran %v, last %+v; want the interrupted night run again", n.ran, n.last)
	}
}

func TestANightThatRecordedNothingWaitsAnHourBeforeTheNextTry(t *testing.T) {
	// invariant: an upgrade that ended without a record (the Machinefile
	// not found) is tried again an hour later, never at every tick.
	n := &nights{clock: &clock{now: at(12, 3, 0)}, bound: "gone", since: at(11, 10, 0)}
	job := n.job(at(11, 10, 0))
	for _, now := range []time.Time{at(12, 3, 0), at(12, 3, 30), at(12, 4, 0)} {
		n.clock.now = now
		job.Run(context.Background())
		n.finish()
	}
	if len(n.ran) != 2 {
		t.Errorf("tried %d times by 04:00; want at 03:00 and 04:00", len(n.ran))
	}
}

func TestAFailedNightIsRecordedAndTriedAgainAnHourLater(t *testing.T) {
	// invariant: a night that failed (its Machinefile does not resolve) is
	// recorded with its error and did not complete: it is tried again an
	// hour later, not at every tick, nor only the next night.
	n := &nights{clock: &clock{now: at(12, 3, 0)}, bound: "laptop", since: at(11, 10, 0), records: true, failure: "laptop does not resolve"}
	job := n.job(at(11, 10, 0))
	for _, now := range []time.Time{at(12, 3, 0), at(12, 3, 30), at(12, 4, 0)} {
		n.clock.now = now
		job.Run(context.Background())
		n.finish()
	}
	if len(n.ran) != 2 || n.last.Error == "" {
		t.Errorf("tried %d times by 04:00, last %+v; want at 03:00 and 04:00, the error recorded", len(n.ran), n.last)
	}
}

func TestNoStepReadsTheLedgerBetweenTwoNights(t *testing.T) {
	// invariant: once a check finds no night due, the steps before the
	// next 03:00 read neither the binding nor the ledger.
	n := &nights{clock: &clock{now: at(12, 4, 0)}, bound: "laptop", since: at(11, 10, 0), records: true}
	job := n.job(at(11, 10, 0))
	for now := at(12, 4, 0); now.Before(at(13, 3, 0)); now = now.Add(10 * time.Minute) {
		n.clock.now = now
		job.Run(context.Background())
		n.finish()
	}
	if len(n.ran) != 1 || n.reads != 2 {
		t.Errorf("ran %v with %d reads of the ledger by 02:50; want one night and two reads", n.ran, n.reads)
	}
	n.clock.now = at(13, 3, 0)
	job.Run(context.Background())
	n.finish()
	if len(n.ran) != 2 {
		t.Errorf("ran %v at the next 03:00; want a second night", n.ran)
	}
}

func TestALedgerThatCannotBeReadIsReadAgainAnHourLaterNotAtEveryStep(t *testing.T) {
	// invariant: a ledger an older station wrote is refused until init
	// brings it to today; meanwhile the steps read it once an hour, and no
	// night runs over it.
	n := &nights{clock: &clock{now: at(12, 3, 0)}, bound: "laptop", since: at(11, 10, 0), unread: errors.New("written at version 1 by an older station")}
	job := n.job(at(11, 10, 0))
	for now := at(12, 3, 0); !now.After(at(12, 4, 0)); now = now.Add(10 * time.Minute) {
		n.clock.now = now
		job.Run(context.Background())
		n.finish()
	}
	if n.reads != 2 || len(n.ran) != 0 {
		t.Errorf("read the ledger %d times and ran %v from 03:00 to 04:00; want two reads, at 03:00 and 04:00, and no night", n.reads, n.ran)
	}
}
