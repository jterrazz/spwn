package machines

import (
	"testing"
	"time"
)

func TestACreateMovesOnStepByStepToItsEnd(t *testing.T) {
	at := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	last := 0
	for _, step := range []string{BornCloning, BornBooting, BornAddress, BornSSH} {
		b := BirthAt(step, at)
		if b.Progress <= last || b.Progress >= 1000 || b.Step != step {
			t.Errorf("%q: progress %d after %d: each step moves the create on, short of done", step, b.Progress, last)
		}
		last = b.Progress
	}
}

func TestAVMBeingCreatedReadsCreatingWhateverItsDiskSays(t *testing.T) {
	at := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	b := BirthAt(BornBooting, at)
	for _, c := range []struct {
		name  string
		vm    VM
		now   time.Time
		state VMState
		birth bool
	}{
		{"its disk stopped while it boots", VM{State: Stopped}, at.Add(5 * time.Second), Creating, true},
		{"its run up while it waits for SSH", VM{State: Running}, at.Add(time.Minute), Creating, true},
		{"its create stalled for half an hour, its disk stopped", VM{State: Stopped}, at.Add(31 * time.Minute), Stopped, false},
		{"its create stalled, its run up", VM{State: Running}, at.Add(31 * time.Minute), Running, false},
	} {
		got := Born(c.vm, b, c.now)
		if got.State != c.state || (got.Birth != nil) != c.birth {
			t.Errorf("%s: %s with birth %v, want %s with birth %v", c.name, got.State, got.Birth != nil, c.state, c.birth)
		}
	}
	if got := Born(VM{State: Stopped}, nil, at); got.State != Stopped {
		t.Errorf("a VM no create runs for reads %s, want its own state", got.State)
	}
}

func TestAVMIsFreshForEightSecondsOnceReady(t *testing.T) {
	ready := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	vm := VM{State: Running, ReadyAt: ready}
	for _, c := range []struct {
		after time.Duration
		fresh bool
	}{{0, true}, {7*time.Second + 900*time.Millisecond, true}, {8 * time.Second, false}, {time.Minute, false}, {-time.Second, false}} {
		if got := Fresh(vm, ready.Add(c.after)); got != c.fresh {
			t.Errorf("%v after ready: fresh %v, want %v", c.after, got, c.fresh)
		}
	}
	if Fresh(VM{State: Running}, ready) {
		t.Error("a VM no create made ready is never fresh")
	}
}
