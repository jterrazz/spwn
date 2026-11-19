package process

import (
	"context"
	"testing"
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

type approval struct {
	answer model.LocalNetwork
	reads  int
}

func (a *approval) read(context.Context) model.LocalNetwork {
	a.reads++
	return a.answer
}

func TestTheEngineReadsLocalNetworkAtItsStartAndEvery30sUntilApproved(t *testing.T) {
	h := newHarness(t)
	a := &approval{answer: model.LocalNetworkNotApproved}
	h.d.o.LocalNetwork = a.read
	if err := h.d.Begin(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	if got := h.d.Hello().LocalNetwork; a.reads != 1 || got != model.LocalNetworkNotApproved {
		t.Fatalf("at its start: %d reads, hello %q", a.reads, got)
	}
	h.clock.At = h.clock.At.Add(29 * time.Second)
	h.d.step(t.Context(), 1)
	if a.reads != 1 {
		t.Errorf("29 s after its start: %d reads, want none again", a.reads)
	}
	a.answer = model.LocalNetworkApproved
	h.clock.At = h.clock.At.Add(time.Second)
	h.d.step(t.Context(), 2)
	if got := h.d.Hello().LocalNetwork; a.reads != 2 || got != model.LocalNetworkApproved {
		t.Fatalf("30 s after its start: %d reads, hello %q", a.reads, got)
	}
	h.clock.At = h.clock.At.Add(time.Minute)
	h.d.step(t.Context(), 3)
	if a.reads != 2 {
		t.Errorf("once approved: %d reads, want no more", a.reads)
	}
}

func TestALocalNetworkReadPastItsSecondCannotTell(t *testing.T) {
	h := newHarness(t)
	h.d.o.LocalNetwork = func(ctx context.Context) model.LocalNetwork {
		<-ctx.Done()
		return model.LocalNetworkNotApproved
	}
	h.d.readLocalNetwork(t.Context())
	if got := h.d.Hello().LocalNetwork; got != "" {
		t.Errorf("a read that did not end: hello %q, want none", got)
	}
}
