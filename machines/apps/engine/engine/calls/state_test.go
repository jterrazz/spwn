package calls

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/store"
)

var t0 = time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)

const own store.Key = "mac:macbook-pro"

// value: an engine's store whose stream gathers changes at once and never
// ticks, so each frame a test reads is one a revision made.
type held struct {
	st *store.Store
	// value: told each time the stream gathers a revision, when not nil.
	gathered chan struct{}
}

func (h held) Store() *store.Store   { return h.st }
func (h held) Owns(store.Key) bool   { return true }
func (h held) Attach() func()        { return func() {} }
func (h held) Ready(context.Context) {}
func (h held) Now() time.Time        { return h.st.Now().At }
func (h held) Sleep(ctx context.Context, d time.Duration) error {
	if d == stateTick {
		<-ctx.Done()
		return ctx.Err()
	}
	if h.gathered != nil {
		h.gathered <- struct{}{}
	}
	return nil
}

// value: one view, `machines`, the This Mac slice's value said as of the
// snapshot's instant only when it holds one, as a view of words would.
type listView struct{}

func (listView) Known(name string) bool { return name == "machines" }

func (listView) Project(s *store.Snapshot, asked func(string) bool) map[string]store.View {
	sl, ok := s.Slices[own]
	if !ok || !asked("machines") {
		return nil
	}
	return map[string]store.View{"machines": {Value: sl.Value, ReadAt: sl.ReadAt, State: sl.State}}
}

func newHeld(t *testing.T) held {
	t.Helper()
	return held{st: store.New(store.EpochOf(t0, 4123), t0, func(error) *fail.Error { return nil }), gathered: make(chan struct{})}
}

func (h held) put(t *testing.T, v any) {
	t.Helper()
	if err := h.st.Put(store.Put{Key: own, Value: v, ReadAt: h.st.Now().At}); err != nil {
		t.Fatal(err)
	}
}

// value: the list view, counting the snapshots it is asked to project.
type countedView struct {
	listView
	projected *atomic.Int32
}

func (c countedView) Project(s *store.Snapshot, asked func(string) bool) map[string]store.View {
	c.projected.Add(1)
	return c.listView.Project(s, asked)
}

func open(t *testing.T, h held, in engine.StateIn) <-chan call.Update[engine.StateFrame] {
	t.Helper()
	return openOn(t, h, Projected(listView{}), in)
}

func openOn(t *testing.T, h held, views *Projections, in engine.StateIn) <-chan call.Update[engine.StateFrame] {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	asked, err := patternsOf(in.Keys, views)
	if err != nil {
		t.Fatal(err)
	}
	out := make(chan call.Update[engine.StateFrame])
	go stream(ctx, h, 5, asked, in, out)
	return out
}

func next(t *testing.T, frames <-chan call.Update[engine.StateFrame]) engine.StateFrame {
	t.Helper()
	wait, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	select {
	case u := <-frames:
		if u.Err != nil {
			t.Fatal(u.Err)
		}
		return u.Value
	case <-wait.Done():
		t.Fatal("no frame")
	}
	return engine.StateFrame{}
}

func TestAViewIsSentOncePerRevisionThatMovesIt(t *testing.T) {
	h := newHeld(t)
	h.put(t, []string{"linux-1"})
	frames := open(t, h, engine.StateIn{Keys: []string{"machines", string(own)}, Revision: engine.StateRevision, Hold: true})
	if f := next(t, frames); f.Event != engine.StateOpened {
		t.Fatalf("first frame %q, want opened", f.Event)
	}
	if f := next(t, frames); f.Event != engine.StateSnapshot || len(f.Slices) != 2 {
		t.Fatalf("second frame %q with %d keys, want a snapshot of the slice and the view", f.Event, len(f.Slices))
	}

	h.st.Clock(t0.Add(store.ClockEvery))
	<-h.gathered
	quiet, stop := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer stop()
	select {
	case u := <-frames:
		t.Fatalf("the clock alone sent %q at %d", u.Value.Event, u.Value.Rev)
	case <-quiet.Done():
	}
	h.put(t, []string{"linux-1", "linux-2"})
	<-h.gathered

	f := next(t, frames)
	want := h.st.Now().Rev
	if f.Event != engine.StateChanged || f.Rev != want {
		t.Fatalf("frame %q at %d, want the change at %d", f.Event, f.Rev, want)
	}
	var got []string
	if err := json.Unmarshal(f.Slices["machines"].Value, &got); err != nil || len(got) != 2 {
		t.Fatalf("the view moved to %s (%v), want both VMs", f.Slices["machines"].Value, err)
	}
}

func TestAResumeOfThisEngineIsTheChangeAndOfAnotherASnapshot(t *testing.T) {
	h := newHeld(t)
	h.put(t, []string{"linux-1"})
	h.st.Clock(t0.Add(store.ClockEvery))
	since := &engine.StateSince{Epoch: string(h.st.Now().Epoch), Rev: h.st.Now().Rev}
	if err := h.st.Put(store.Put{Key: "ports:macbook-pro", Value: 1, ReadAt: t0}); err != nil {
		t.Fatal(err)
	}
	keys := []string{"machines", "local"}

	h.gathered = nil
	resumed := open(t, h, engine.StateIn{Keys: keys, Revision: engine.StateRevision, Since: since})
	next(t, resumed)
	changed := next(t, resumed)
	other := open(t, h, engine.StateIn{Keys: keys, Revision: engine.StateRevision, Since: &engine.StateSince{Epoch: "2026-11-20T09:00:00Z/1", Rev: since.Rev}})
	next(t, other)
	snapshot := next(t, other)

	if _, moved := changed.Slices["ports:macbook-pro"]; changed.Event != engine.StateChanged || !moved || changed.Slices["machines"].Value == nil || len(changed.Slices) != 2 {
		t.Fatalf("resume %q with %v, want the ports change and the view whole", changed.Event, changed.Slices)
	}
	if snapshot.Event != engine.StateSnapshot || len(snapshot.Slices) != 3 {
		t.Fatalf("another engine's resume %q with %d keys, want a snapshot of three", snapshot.Event, len(snapshot.Slices))
	}
}

func TestAClientOfAnotherRevisionIsAnsweredItsVerdictWhateverKeysItAsks(t *testing.T) {
	h := newHeld(t)
	views := Projected(listView{})
	keys := []string{"machines", "engine.a-later-view"}
	for _, c := range []struct {
		revision int
		want     engine.FitVerdict
	}{{engine.StateRevision + 1, engine.EngineOlder}, {engine.StateRevision - 1, engine.AppOlder}, {0, engine.AppOlder}} {
		in := engine.StateIn{Keys: keys, Revision: c.revision, Hold: true}
		asked, err := judgedOf(in, views)
		if err != nil {
			t.Fatalf("a client of revision %d was refused %v, want its verdict", c.revision, err)
		}
		ctx, cancel := context.WithCancel(t.Context())
		out := make(chan call.Update[engine.StateFrame])
		go stream(ctx, h, 5, asked, in, out)
		opened := next(t, out)
		if opened.Event != engine.StateOpened || opened.Fit == nil || opened.Fit.Verdict != c.want {
			t.Fatalf("a client of revision %d opened %+v, want the verdict %s", c.revision, opened, c.want)
		}
		if _, more := <-out; more {
			t.Fatalf("a client of revision %d was sent a frame after its verdict", c.revision)
		}
		cancel()
	}

	_, err := judgedOf(engine.StateIn{Keys: keys, Revision: engine.StateRevision}, views)
	if e := new(fail.Error); !errors.As(err, &e) || e.Code != StateUnknownKey {
		t.Fatalf("a fitting client asking an unknown view got %v, want %s", err, StateUnknownKey)
	}
}

func TestTwoHeldStreamsProjectOneRevisionOnce(t *testing.T) {
	h := newHeld(t)
	h.put(t, []string{"linux-1"})
	var projected atomic.Int32
	views := Projected(countedView{projected: &projected})
	in := engine.StateIn{Keys: []string{"machines"}, Revision: engine.StateRevision, Hold: true}
	first, second := openOn(t, h, views, in), openOn(t, h, views, in)
	for _, frames := range []<-chan call.Update[engine.StateFrame]{first, second} {
		next(t, frames)
		next(t, frames)
	}

	h.put(t, []string{"linux-1", "linux-2"})
	<-h.gathered
	<-h.gathered
	a, b := next(t, first), next(t, second)

	if a.Rev != b.Rev || string(a.Slices["machines"].Value) != string(b.Slices["machines"].Value) {
		t.Fatalf("the streams sent %s at %d and %s at %d, want one projection", a.Slices["machines"].Value, a.Rev, b.Slices["machines"].Value, b.Rev)
	}
	if n := projected.Load(); n != 2 {
		t.Fatalf("two revisions projected %d times for two streams, want once each", n)
	}
}
