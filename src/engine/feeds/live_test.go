package feeds_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/activity"
	activitycalls "github.com/jterrazz/studio-station/src/activity/calls"
	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/engine"
	"github.com/jterrazz/studio-station/src/engine/feeds"
	"github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
)

type moving struct {
	mu  sync.Mutex
	now time.Time
}

func (m *moving) Sleep(ctx context.Context, _ time.Duration) error {
	<-ctx.Done()
	return ctx.Err()
}

func (m *moving) Now() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.now
}

func (m *moving) pass(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.now = m.now.Add(d)
}

// value: the Macs' samples, This Mac answering at once and the mini only
// while it is not late.
type fleet struct {
	mu    sync.Mutex
	late  chan struct{}
	asked map[string]uint64
	at    map[string]time.Time
}

func (f *fleet) Invoke(ctx context.Context, target machine.Ref, name string, _, out any, _ call.Progress) error {
	f.mu.Lock()
	at := target.Name
	if at == "" {
		at = thisMac
	}
	f.asked[at]++
	f.at[at] = f.at[at].Add(2 * time.Second)
	busy, when, late := f.asked[at], f.at[at], f.late
	f.mu.Unlock()
	if at == mini && late != nil {
		select {
		case <-late:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if name != activitycalls.Sample.Name {
		return nil
	}
	memory := map[string]int64{thisMac: 16 * gb, mini: 64 * gb}[at]
	r, ok := out.(*activity.Reading)
	if !ok {
		return errors.New("a sample answers a reading")
	}
	*r = activity.Reading{Sample: activity.Sample{At: when, CPUBusy: busy * 300, CPUTotal: busy * 1000, MemUsed: memory / 4, MemTotal: memory}}
	return nil
}

func (f *fleet) Watch(context.Context, machine.Ref, string, any, call.Progress) (<-chan call.Update[any], error) {
	return nil, nil
}

func (f *fleet) lateFromNow() func() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.late = make(chan struct{})
	late := f.late
	var once sync.Once
	return func() {
		once.Do(func() {
			f.mu.Lock()
			f.late = nil
			f.mu.Unlock()
			close(late)
		})
	}
}

func lead(s engine.Sample, key string) page.Value {
	v, _ := s.Values[key].(page.Value)
	if v.Kind == page.KindSpark {
		lead := v.Items[0]
		lead.Stale = v.Stale
		return lead
	}
	return v
}

func TestALateRemoteMacKeepsItsLastValueTurnsStaleAndNeverHoldsTheTick(t *testing.T) {
	f, clock := &fleet{asked: map[string]uint64{}, at: map[string]time.Time{}}, &moving{now: time.Date(2026, 11, 3, 10, 30, 0, 0, time.UTC)}
	live := &feeds.Live{Calls: f, Clock: clock, ThisMac: thisMac, Budget: 50 * time.Millisecond}
	keys := []string{"cpu:" + thisMac, "memory:" + mini, "free:" + mini}
	live.Sample(t.Context(), keys)
	before := live.Sample(t.Context(), keys)
	if got := lead(before, "memory:"+mini); got.Kind != page.KindRatio || got.Of != 64*gb || got.Stale {
		t.Fatalf("the mini answering gives its memory %+v, want its own 64 GB, fresh", got)
	}
	answer := f.lateFromNow()
	defer answer()
	clock.pass(10 * time.Second)
	late := func() engine.Sample { return live.Sample(t.Context(), keys) }
	during := late()
	if got, was := lead(during, "memory:"+mini), lead(before, "memory:"+mini); got.N != was.N || got.Of != was.Of || got.Stale {
		t.Errorf("the mini late 10 s gives its memory %+v, want its last value %+v, not yet stale", got, lead(before, "memory:"+mini))
	}
	if got := lead(during, "cpu:"+thisMac); got.Kind != page.KindPercent {
		t.Errorf("This Mac's CPU is %+v while the mini is late, want This Mac read as ever", got)
	}
	clock.pass(25 * time.Second)
	stale := late()
	if got := lead(stale, "free:"+mini); !got.Stale || got.N != 48*gb {
		t.Errorf("the mini unheard for 35 s shows its free memory %+v, want its last 48 GB, stale", got)
	}
	if want := []string{"memory:" + mini, "free:" + mini}; !slices.Equal(stale.Stale, want) {
		t.Errorf("the tick names the stale keys %q, want the mini's %q alone", stale.Stale, want)
	}
	f.mu.Lock()
	asked := f.asked[mini]
	f.mu.Unlock()
	if asked != 3 {
		t.Errorf("the mini was asked %d times, want once more while late, never a second ask beside the one it owes", asked)
	}
	answer()
	for range 50 {
		if back := live.Sample(t.Context(), keys); len(back.Stale) == 0 {
			return
		}
	}
	t.Errorf("the mini answering again stays stale, want it fresh")
}

func TestHomeAndAVMPageListTheKeysTheirMovingNumbersFollow(t *testing.T) {
	w := &world{infra: []machines.VM{infraAt(machines.Running, 0)}, asked: map[string][]string{}, activity: map[string]uint64{}}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	o := feeds.Menubar{Calls: w, Clock: newSteps(), ThisMachine: page.Context{Machine: thisMac}, Report: func(awake.View) *awake.Report { return nil }}
	home := until(t, o.Feed()(ctx, engine.FeedInput{Changed: make(chan struct{}), Refresh: make(chan struct{}), Awake: func() awake.View { return awake.View{} }}), "home", func(p page.Page) bool { return p.Strip != nil })
	if want := []string{"cpu:" + thisMac, "memory:" + thisMac}; !slices.Equal(home.Live, want) {
		t.Errorf("home reads the live keys %q, want This Mac's %q", home.Live, want)
	}
	if home.Strip.CPU.Live != "cpu:"+thisMac {
		t.Errorf("home's CPU follows %q, want This Mac's CPU key", home.Strip.CPU.Live)
	}
}
