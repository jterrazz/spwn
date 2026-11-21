package store

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
)

const (
	// value: how often the store's clock advances, so a page's relative words
	// ("Not seen for 12 min") are projected again.
	ClockEvery = 15 * time.Second
	// value: how many revisions Since can answer from; an older one is a
	// snapshot.
	kept = 256
)

type Put struct {
	Key    Key
	Value  any
	ReadAt time.Time
}

type change struct {
	rev     uint64
	changed []Key
	gone    []Key
}

type Store struct {
	word  call.Wording
	epoch Epoch

	mu   sync.Mutex
	held map[Key]Slice
	// invariant: a revision's Snapshot never changes once published: a reading
	// that leaves a value as it was moves only the held ReadAt, which the key's
	// next change publishes, so a projection memoised per revision stays true.
	now      *Snapshot
	ring     [kept]change
	watchers map[chan *Snapshot]struct{}
}

func New(epoch Epoch, at time.Time, word call.Wording) *Store {
	return &Store{
		word: word, epoch: epoch,
		held:     map[Key]Slice{},
		now:      &Snapshot{Epoch: epoch, At: at, Slices: map[Key]Slice{}},
		watchers: map[chan *Snapshot]struct{}{},
	}
}

func (s *Store) Now() *Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.now
}

func (s *Store) Put(puts ...Put) error {
	type encoded struct {
		Put
		raw  []byte
		hash string
	}
	all := make([]encoded, 0, len(puts))
	for _, p := range puts {
		if _, err := ParseKey(string(p.Key)); err != nil {
			return err
		}
		raw, err := call.Encode(p.Value, s.word)
		if err != nil {
			return fmt.Errorf("%s: %w", p.Key, err)
		}
		all = append(all, encoded{Put: p, raw: raw, hash: hashOf(raw)})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var changed []Key
	for _, p := range all {
		old, had := s.held[p.Key]
		if had && old.State == Fresh && old.Hash == p.hash {
			old.Value, old.ReadAt = p.Value, p.ReadAt
			s.held[p.Key] = old
			continue
		}
		s.held[p.Key] = Slice{Key: p.Key, Value: p.Value, Raw: p.raw, Hash: p.hash, ReadAt: p.ReadAt, State: Fresh}
		changed = append(changed, p.Key)
	}
	s.commit(changed, nil, s.now.At)
	return nil
}

func (s *Store) Fail(key Key, why string) error {
	// invariant: a failed reading keeps the value it had, marked stale; a key
	// never read stays unread and only learns why.
	if _, err := ParseKey(string(key)); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	old, had := s.held[key]
	if !had {
		old = Slice{Key: key, State: Unread}
	}
	next := old
	if next.State == Fresh {
		next.State = Stale
	}
	next.Why = why
	if had && next.State == old.State && next.Why == old.Why {
		return nil
	}
	s.held[key] = next
	s.commit([]Key{key}, nil, s.now.At)
	return nil
}

func (s *Store) Drop(key Key) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, had := s.held[key]; !had {
		return
	}
	delete(s.held, key)
	s.commit(nil, []Key{key}, s.now.At)
}

func (s *Store) Sweep(now time.Time, cadence func(Key) (Cadence, bool)) {
	// invariant: cadence answers false for a slice read on change only, which
	// no time turns stale.
	s.mu.Lock()
	defer s.mu.Unlock()
	var changed []Key
	for _, k := range slices.Sorted(maps.Keys(s.held)) {
		sl := s.held[k]
		c, ok := cadence(k)
		if !ok || sl.State != Fresh || now.Sub(sl.ReadAt) <= c.staleAfter() {
			continue
		}
		sl.State, sl.Why = Stale, WhyLate
		s.held[k] = sl
		changed = append(changed, k)
	}
	s.commit(changed, nil, s.now.At)
}

func (s *Store) Clock(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if now.Sub(s.now.At) < ClockEvery {
		return
	}
	s.commit(nil, nil, now)
}

func (s *Store) commit(changed, gone []Key, at time.Time) {
	if len(changed) == 0 && len(gone) == 0 && at.Equal(s.now.At) {
		return
	}
	rev := s.now.Rev + 1
	next := &Snapshot{Epoch: s.epoch, Rev: rev, At: at, Slices: maps.Clone(s.now.Slices)}
	for _, k := range changed {
		sl := s.held[k]
		sl.Rev = rev
		s.held[k] = sl
		next.Slices[k] = sl
	}
	for _, k := range gone {
		delete(next.Slices, k)
	}
	s.ring[rev%kept] = change{rev: rev, changed: changed, gone: gone}
	s.now = next
	for c := range s.watchers {
		select {
		case <-c:
		default:
		}
		c <- next
	}
}

func (s *Store) Since(epoch Epoch, rev uint64) (changed, gone []Key, ok bool) {
	// invariant: ok is false for another engine's revision, one ahead of this
	// store, or one older than the ring: the asker takes a snapshot.
	s.mu.Lock()
	defer s.mu.Unlock()
	cur := s.now.Rev
	if epoch != s.epoch || rev > cur || cur-rev > kept {
		return nil, nil, false
	}
	moved, left := map[Key]bool{}, map[Key]bool{}
	for r := rev + 1; r <= cur; r++ {
		c := s.ring[r%kept]
		for _, k := range c.changed {
			moved[k] = true
			delete(left, k)
		}
		for _, k := range c.gone {
			left[k] = true
			delete(moved, k)
		}
	}
	return slices.Sorted(maps.Keys(moved)), slices.Sorted(maps.Keys(left)), true
}

func (s *Store) Watch(ctx context.Context) <-chan *Snapshot {
	// invariant: latest-only: a watcher that does not read misses revisions,
	// never blocks a writer; it is handed the current snapshot at once.
	c := make(chan *Snapshot, 1)
	s.mu.Lock()
	c <- s.now
	s.watchers[c] = struct{}{}
	s.mu.Unlock()
	go func() {
		<-ctx.Done()
		s.mu.Lock()
		delete(s.watchers, c)
		close(c)
		s.mu.Unlock()
	}()
	return c
}
