package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/engine"
)

type asks struct {
	clock *clock
	kept  *engine.ReleaseKept
	down  bool
	n     int
	held  []func()
}

func (a *asks) job() *Release {
	return &Release{
		Clock: a.clock,
		Kept: func(context.Context) (engine.ReleaseKept, bool, error) {
			if a.kept == nil {
				return engine.ReleaseKept{}, false, nil
			}
			return *a.kept, true, nil
		},
		Refresh: func(context.Context) error {
			a.n++
			k := engine.ReleaseKept{}
			if a.kept != nil {
				k = *a.kept
			}
			if a.down {
				k = k.Failed(a.clock.Now(), "GitHub said 502")
				a.kept = &k
				return errors.New("GitHub said 502")
			}
			k = k.Answered(a.clock.Now(), engine.Tagged{}, "arm64")
			a.kept = &k
			return nil
		},
		Go: func(f func()) { a.held = append(a.held, f) },
	}
}

func (a *asks) step(j *Release) {
	j.Run(context.Background())
	for _, f := range a.held {
		f()
	}
	a.held = nil
}

func TestTheReleaseIsAskedAtTheFirstStepThenEverySixHours(t *testing.T) {
	a := &asks{clock: &clock{now: time.Date(2026, 11, 19, 8, 0, 0, 0, time.UTC)}}
	j := a.job()
	a.step(j)
	if a.n != 1 {
		t.Fatalf("asked %d times at the first step; want 1", a.n)
	}
	for _, d := range []time.Duration{time.Second, 30 * time.Minute, 6*time.Hour - time.Second} {
		a.clock.now = time.Date(2026, 11, 19, 8, 0, 0, 0, time.UTC).Add(d)
		a.step(j)
	}
	if a.n != 1 {
		t.Fatalf("asked %d times within 6 h of an answer; want 1", a.n)
	}
	a.clock.now = time.Date(2026, 11, 19, 14, 0, 0, 0, time.UTC)
	a.step(j)
	if a.n != 2 {
		t.Errorf("asked %d times 6 h after the answer; want 2", a.n)
	}
}

func TestARefusedAskIsTriedAgainHalfAnHourLater(t *testing.T) {
	start := time.Date(2026, 11, 19, 8, 0, 0, 0, time.UTC)
	a := &asks{clock: &clock{now: start}, down: true}
	j := a.job()
	a.step(j)
	a.clock.now = start.Add(30*time.Minute - time.Second)
	a.step(j)
	if a.n != 1 {
		t.Fatalf("asked %d times within 30 min of a refusal; want 1", a.n)
	}
	a.clock.now = start.Add(30 * time.Minute)
	a.step(j)
	if a.n != 2 {
		t.Errorf("asked %d times 30 min after a refusal; want 2", a.n)
	}
}

func TestAnEngineStartedAgainAsksNothingBeforeItsTime(t *testing.T) {
	read := time.Date(2026, 11, 19, 8, 0, 0, 0, time.UTC)
	a := &asks{clock: &clock{now: read.Add(time.Hour)}, kept: &engine.ReleaseKept{Read: read}}
	a.step(a.job())
	if a.n != 0 {
		t.Errorf("asked %d times an hour after the kept answer; want none", a.n)
	}
}

func TestOneAskAtATime(t *testing.T) {
	a := &asks{clock: &clock{now: time.Date(2026, 11, 19, 8, 0, 0, 0, time.UTC)}}
	j := a.job()
	j.Run(context.Background())
	j.Run(context.Background())
	if len(a.held) != 1 {
		t.Errorf("%d asks started while one runs; want 1", len(a.held))
	}
}
