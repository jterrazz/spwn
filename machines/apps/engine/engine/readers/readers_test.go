package readers_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/engine/readers"
	"github.com/jterrazz/studio-station/apps/engine/engine/store"
)

type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

func (c *clock) Sleep(ctx context.Context, _ time.Duration) error { return ctx.Err() }

func word(err error) *fail.Error { return &fail.Error{Message: err.Error()} }

type counted struct {
	mu   sync.Mutex
	n    int
	fail bool
}

func (c *counted) reader(key store.Key) *readers.Reader {
	return &readers.Reader{Name: string(key), Idle: time.Minute, Attached: 30 * time.Second, Budget: 5 * time.Second,
		Read: func(context.Context, *store.Snapshot) readers.Reading {
			c.mu.Lock()
			defer c.mu.Unlock()
			c.n++
			if c.fail {
				return readers.Reading{Owns: []store.Key{key}, Failed: map[store.Key]string{key: store.WhyLate}}
			}
			return readers.Reading{Owns: []store.Key{key}, Puts: []store.Put{{Key: key, Value: map[string]int{"vms": 2}}}}
		}}
}

func (c *counted) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

func TestOneReadPerPeriodWhateverTheSubscribers(t *testing.T) {
	c := &clock{now: time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)}
	st := store.New(store.EpochOf(c.now, 1), c.now, word)
	reads := &counted{}
	key := store.Of(store.FamilyMac, "macbook-pro")
	set := readers.New(readers.Options{Store: st, Clock: c, Run: func(f func()) { f() }, Readers: []*readers.Reader{reads.reader(key)}})
	releases := make([]func(), 0, 5)
	for range 5 {
		releases = append(releases, set.Attach())
	}
	defer func() {
		for _, release := range releases {
			release()
		}
	}()
	for range 30 {
		set.Tick(t.Context(), false)
		c.advance(time.Second)
	}
	if got := reads.count(); got != 1 {
		t.Fatalf("30 s attached, five subscribers: %d reads, want 1", got)
	}
	set.Tick(t.Context(), false)
	if got := reads.count(); got != 2 {
		t.Fatalf("at the attached period: %d reads, want 2", got)
	}
	if !set.Owns(key) || st.Now().Slices[key].State != store.Fresh {
		t.Fatalf("the slice is not held fresh: %+v", st.Now().Slices[key])
	}
}

func TestAFailedReadingKeepsItsValueStale(t *testing.T) {
	c := &clock{now: time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)}
	st := store.New(store.EpochOf(c.now, 1), c.now, word)
	reads := &counted{}
	key := store.Of(store.FamilyMac, "macbook-pro")
	set := readers.New(readers.Options{Store: st, Clock: c, Run: func(f func()) { f() }, Readers: []*readers.Reader{reads.reader(key)}})
	set.Tick(t.Context(), true)
	held := st.Now().Slices[key]
	reads.fail = true
	c.advance(30 * time.Second)
	set.Tick(t.Context(), true)
	got := st.Now().Slices[key]
	if got.State != store.Stale || got.Why != store.WhyLate || got.Hash != held.Hash || got.Value == nil {
		t.Fatalf("a late reading: %+v, want the value of %s kept, stale and late", got, held.Hash)
	}
}

func TestASliceUnreadPastItsPeriodTurnsStale(t *testing.T) {
	c := &clock{now: time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)}
	st := store.New(store.EpochOf(c.now, 1), c.now, word)
	reads := &counted{}
	key := store.Of(store.FamilyMac, "macbook-pro")
	running := false
	set := readers.New(readers.Options{Store: st, Clock: c, Run: func(f func()) {
		if !running {
			running = true
			f()
		}
	}, Readers: []*readers.Reader{reads.reader(key)}})
	set.Tick(t.Context(), false)
	c.advance(2*time.Minute + 5*time.Second + time.Second)
	set.Tick(t.Context(), false)
	if got := st.Now().Slices[key]; got.State != store.Stale || got.Why != store.WhyLate {
		t.Fatalf("idle, unread for twice its period plus its budget: %+v, want stale", got)
	}
}
