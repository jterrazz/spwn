package process

import (
	"context"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/approvals"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

type approval struct {
	answer approvals.State
	reads  int
}

func (a *approval) read(context.Context) approvals.State {
	a.reads++
	return a.answer
}

func (h harness) after(t *testing.T, d time.Duration, tick int) {
	t.Helper()
	h.clock.At = h.clock.At.Add(d)
	h.d.step(t.Context(), tick)
}

func TestARefusalInTheEnginesFirst90sReadsCheckingAndIsReadAgainEvery2s(t *testing.T) {
	h := newHarness(t)
	a := &approval{answer: approvals.Refused}
	h.d.o.LocalNetwork = a.read
	if err := h.d.Begin(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	if got := h.d.Hello().LocalNetwork; a.reads != 1 || got != model.LocalNetworkChecking {
		t.Fatalf("at its start: %d reads, hello %q, want 1 read and %q", a.reads, got, model.LocalNetworkChecking)
	}
	h.after(t, time.Second, 1)
	if a.reads != 1 {
		t.Errorf("1 s after its start: %d reads, want none again", a.reads)
	}
	a.answer = approvals.Approved
	h.after(t, time.Second, 2)
	if got := h.d.Hello().LocalNetwork; a.reads != 2 || got != model.LocalNetworkApproved {
		t.Fatalf("2 s after its start: %d reads, hello %q, want 2 reads and approved", a.reads, got)
	}
	h.after(t, time.Minute, 3)
	if a.reads != 2 {
		t.Errorf("once approved: %d reads, want no more", a.reads)
	}
}

func TestARefusalPast90sReadsNotApprovedThenEvery30s(t *testing.T) {
	h := newHarness(t)
	a := &approval{answer: approvals.Refused}
	h.d.o.LocalNetwork = a.read
	if err := h.d.Begin(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	for tick := 1; tick <= 44; tick++ {
		h.after(t, 2*time.Second, tick)
		if got := h.d.Hello().LocalNetwork; got != model.LocalNetworkChecking {
			t.Fatalf("%d s after its start: hello %q, want %q", 2*tick, got, model.LocalNetworkChecking)
		}
	}
	if a.reads != 45 {
		t.Errorf("over its first 88 s: %d reads, want 45, one every 2 s", a.reads)
	}
	h.after(t, 2*time.Second, 45)
	if got := h.d.Hello().LocalNetwork; got != model.LocalNetworkNotApproved {
		t.Fatalf("90 s after its start: hello %q, want %q", got, model.LocalNetworkNotApproved)
	}
	h.after(t, 29*time.Second, 46)
	if a.reads != 46 {
		t.Errorf("29 s after its verdict: %d reads, want none again", a.reads)
	}
	a.answer = approvals.Approved
	h.after(t, time.Second, 47)
	if got := h.d.Hello().LocalNetwork; a.reads != 47 || got != model.LocalNetworkApproved {
		t.Fatalf("30 s after its verdict: %d reads, hello %q", a.reads, got)
	}
}

func TestAnEngineBegunPast90sAfterItsStartReadsARefusalAsNotApproved(t *testing.T) {
	h := newHarness(t)
	h.d.o.LocalNetwork = (&approval{answer: approvals.Refused}).read
	h.d.o.StartedAt = h.clock.At.Add(-2 * time.Minute)
	if err := h.d.Begin(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	if got := h.d.Hello().LocalNetwork; got != model.LocalNetworkNotApproved {
		t.Errorf("begun 2 min after its start: hello %q, want %q", got, model.LocalNetworkNotApproved)
	}
}

func TestALocalNetworkReadPastItsSecondCannotTell(t *testing.T) {
	h := newHarness(t)
	h.d.o.LocalNetwork = func(ctx context.Context) approvals.State {
		<-ctx.Done()
		return approvals.Refused
	}
	h.d.readLocalNetwork(t.Context())
	if got := h.d.Hello().LocalNetwork; got != "" {
		t.Errorf("a read that did not end: hello %q, want none", got)
	}
}
