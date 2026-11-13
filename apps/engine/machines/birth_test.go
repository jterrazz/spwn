package machines

import (
	"testing"
	"time"
)

func TestACreateMovesOnStepByStepToItsEnd(t *testing.T) {
	at := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	last := 0
	for i, step := range []BirthStep{BornBuilding, BornCloning, BornBooting, BornAddress, BornSSH} {
		b := BirthAt(step, true, at)
		if b.Progress <= last || b.Progress >= 1000 || b.Step != step || b.Position != i+1 || b.Steps != 5 || !b.Known() {
			t.Errorf("%q: progress %d after %d, %d of %d: each step moves the create on, short of done", step, b.Progress, last, b.Position, b.Steps)
		}
		last = b.Progress
	}
	for i, step := range []BirthStep{BornCloning, BornBooting, BornAddress, BornSSH} {
		if b := BirthAt(step, false, at); b.Position != i+1 || b.Steps != 4 || !b.Known() {
			t.Errorf("%q with no build: %d of %d, want %d of 4", step, b.Position, b.Steps, i+1)
		}
	}
	if b := BirthAt(BornBuilding, false, at); b.Known() {
		t.Errorf("a build in a create that builds nothing reads %d of %d, known", b.Position, b.Steps)
	}
}

func TestAVMBeingCreatedReadsCreatingWhileTheProcessThatRunsItsCreateRuns(t *testing.T) {
	at := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	b := BirthAt(BornBooting, false, at)
	for _, c := range []struct {
		name  string
		vm    VM
		birth *Birth
		runs  bool
		state VMState
		born  bool
	}{
		{"its disk stopped while it boots", VM{State: Stopped}, b, true, Creating, true},
		{"its run up while it waits for SSH", VM{State: Running}, b, true, Creating, true},
		{"its process ended, its disk stopped", VM{State: Stopped}, b, false, Stopped, false},
		{"its process ended, its run up", VM{State: Running}, b, false, Running, false},
		{"a step this station does not know", VM{State: Stopped}, &Birth{Progress: 50, Step: "cloning the image", Position: 1, Steps: 4, At: at}, true, Stopped, false},
		{"a step with no position among the steps", VM{State: Stopped}, &Birth{Progress: 50, Step: BornCloning, At: at}, true, Stopped, false},
	} {
		got := Born(c.vm, c.birth, c.runs)
		if got.State != c.state || (got.Birth != nil) != c.born {
			t.Errorf("%s: %s with birth %v, want %s with birth %v", c.name, got.State, got.Birth != nil, c.state, c.born)
		}
		if !c.born && c.state == Stopped && got.Detail != HalfMade {
			t.Errorf("%s: detail %q, want %q", c.name, got.Detail, HalfMade)
		}
	}
	if got := Born(VM{State: Stopped}, nil, true); got.State != Stopped || got.Detail != "" {
		t.Errorf("a VM no create runs for reads %s %q, want its own state", got.State, got.Detail)
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

func TestABirthSaysWhatItDoesAfterCreating(t *testing.T) {
	for _, c := range []struct {
		birth Birth
		want  string
	}{
		{*BirthAt(BornBuilding, true, time.Time{}), "image 1 of 5"},
		{*BirthAt(BornCloning, true, time.Time{}), "disk 2 of 5"},
		{*BirthAt(BornCloning, false, time.Time{}), "disk 1 of 4"},
		{*BirthAt(BornAddress, false, time.Time{}), "network 3 of 4"},
		{*BirthAt(BornSSH, false, time.Time{}), "SSH 4 of 4"},
	} {
		if got := c.birth.Doing(); got != c.want {
			t.Errorf("%s, %d of %d, reads %q, want %q", c.birth.Step, c.birth.Position, c.birth.Steps, got, c.want)
		}
	}
	if w := BornSSH.Word(); w != "Creating" {
		t.Errorf("a create's step reads %q, want Creating", w)
	}
}
