package readers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
	"github.com/jterrazz/studio-station/apps/engine/engine/store"
)

const (
	// value: a held stream that sends no frame for three of its ticks is
	// lost (ADR-070).
	farSilent = 45 * time.Second
	farWatch  = 15 * time.Second
	farBudget = 5 * time.Second
)

// value: how long a lost stream waits before it is dialed again, by how
// many times in a row it was lost: 5 s, 10 s, 30 s, then every minute.
func farWait(fails int) time.Duration {
	switch fails {
	case 0:
		return 5 * time.Second
	case 1:
		return 10 * time.Second
	case 2:
		return 30 * time.Second
	}
	return time.Minute
}

type Far struct {
	// value: the Router another Mac's engine.state is held through, over
	// the link it keeps to that Mac.
	Calls call.Router
	Store *store.Store
	Clock Clock
	// value: every other Mac of this Mac's records, read now.
	Macs func() []string
	// value: lets go of the link to a Mac, so the next watch dials it
	// again; nil keeps it.
	Drop func(mac string)
	// value: this Mac's own pace, which a held stream tells the other Mac.
	Pace func() Pace
	// value: the context a held stream of another Mac runs in, made from
	// the one that opens it; nil keeps that one.
	Following func(context.Context) context.Context

	mu    sync.Mutex
	links map[string]*link
	held  engine.Held
}

type link struct {
	cancel  context.CancelFunc
	running bool
	pace    Pace
	since   *engine.StateSince
	fails   int
	retry   time.Time
	keys    map[store.Key]bool
	heard   time.Time
	self    string
	// value: closed once the stream's first snapshot is copied, or it
	// failed.
	first chan struct{}
}

func (f *Far) Remember(held engine.Held) {
	// invariant: Remember comes before the first tick: what an engine before
	// this one copied of each Mac is held aside, and comes back only if that
	// Mac does not answer, kept as it was and marked stale; a Mac that
	// answers is copied from its own snapshot, so an engine that remembers
	// counts the revisions of one that does not.
	f.mu.Lock()
	defer f.mu.Unlock()
	f.held = maps.Clone(held)
}

func (f *Far) Held() engine.Held {
	// value: what this engine copied of each other Mac, as Remember takes it.
	now := f.Store.Now()
	f.mu.Lock()
	defer f.mu.Unlock()
	out := engine.Held{}
	for mac, l := range f.links {
		for k := range l.keys {
			sl, ok := now.Slices[k]
			if !ok || sl.State == store.Unread {
				continue
			}
			if out[mac] == nil {
				out[mac] = map[string]engine.SliceOut{}
			}
			out[mac][string(k)] = engine.SliceOut{Value: sl.Raw, Hash: sl.Hash, ReadAt: stamp(sl.ReadAt), State: string(sl.State), Why: sl.Why}
		}
	}
	for mac, kept := range f.held {
		if out[mac] == nil {
			out[mac] = kept
		}
	}
	return out
}

func (f *Far) linkLocked(mac string) *link {
	if f.links == nil {
		f.links = map[string]*link{}
	}
	l, ok := f.links[mac]
	if !ok {
		l = &link{keys: map[store.Key]bool{}}
		f.links[mac] = l
	}
	return l
}

func (f *Far) Tick(ctx context.Context) {
	// invariant: ADR-072: every other Mac's engine.state is held open, and
	// one that stopped or never opened is dialed again once its wait is
	// over; a held stream is opened again, from the revision it holds, when
	// this Mac's pace moves, so the other Mac reads at the pace of whoever
	// reads it. A Mac gone from the records takes its slices with it.
	now := f.Clock.Now()
	pace := f.Pace()
	macs := f.Macs()
	f.mu.Lock()
	var start []string
	for _, mac := range macs {
		l := f.linkLocked(mac)
		if l.running && l.pace != pace {
			l.cancel()
			l.running = false
			l.retry = time.Time{}
		}
		if !l.running && !now.Before(l.retry) {
			start = append(start, mac)
		}
	}
	for mac := range f.held {
		if !slices.Contains(macs, mac) {
			delete(f.held, mac)
		}
	}
	var gone []store.Key
	for mac, l := range f.links {
		if slices.Contains(macs, mac) {
			continue
		}
		if l.running {
			l.cancel()
		}
		gone = append(gone, slices.Collect(maps.Keys(l.keys))...)
		delete(f.links, mac)
	}
	f.mu.Unlock()
	for _, k := range gone {
		f.Store.Drop(k)
	}
	for _, mac := range start {
		f.open(ctx, mac, pace)
	}
}

func (f *Far) Warm(ctx context.Context) {
	// invariant: a stepped engine waits, once, for every other Mac's first
	// snapshot or its failure, each bound by the budget of an ask, so its
	// command's first frame reads what those Macs said.
	f.Tick(ctx)
	f.mu.Lock()
	var firsts []chan struct{}
	for _, l := range f.links {
		if l.running {
			firsts = append(firsts, l.first)
		}
	}
	f.mu.Unlock()
	bound, stop := context.WithTimeout(ctx, farBudget)
	defer stop()
	for _, first := range firsts {
		select {
		case <-first:
		case <-bound.Done():
			return
		}
	}
}

func (f *Far) open(ctx context.Context, mac string, pace Pace) {
	// invariant: another Mac's stream is answered by that Mac's engine,
	// never by the store the connection that opens it carries.
	ctx = enginecalls.WithoutState(ctx)
	if f.Following != nil {
		ctx = f.Following(ctx)
	}
	ctx, cancel := context.WithCancel(ctx)
	f.mu.Lock()
	l := f.linkLocked(mac)
	l.cancel, l.running, l.pace, l.heard = cancel, true, pace, f.Clock.Now()
	l.first = make(chan struct{})
	in := engine.StateIn{Keys: []string{string(store.Local)}, Hold: true, Since: l.since, Revision: engine.StateRevision, Pace: paceWord(pace)}
	first := l.first
	f.mu.Unlock()
	updates, err := call.Follow(ctx, f.Calls, machine.Ref{Name: mac}, enginecalls.StateWatch, in, nil)
	if err != nil {
		f.lost(mac, l, Why(err))
		close(first)
		return
	}
	go f.follow(ctx, mac, l, updates, sync.OnceFunc(func() { close(first) }))
	go f.watchdog(ctx, mac, l)
}

func paceWord(p Pace) string {
	if p == Attached {
		return engine.PaceAttached
	}
	return engine.PaceIdle
}

func (f *Far) follow(ctx context.Context, mac string, l *link, updates <-chan call.Update[engine.StateFrame], first func()) {
	defer first()
	why := store.WhyUnreachable
	for u := range updates {
		if u.Err != nil {
			why = Why(u.Err)
			break
		}
		fr := u.Value
		f.mu.Lock()
		l.heard = f.Clock.Now()
		f.mu.Unlock()
		switch fr.Event {
		case engine.StateOpened:
			if fr.Fit == nil || fr.Fit.Verdict != engine.Fits {
				f.lost(mac, l, store.WhyFarTooOld)
				return
			}
			f.mu.Lock()
			if l.since == nil || l.since.Epoch != fr.Epoch {
				l.since = &engine.StateSince{Epoch: fr.Epoch}
			}
			f.mu.Unlock()
		case engine.StateSnapshot, engine.StateChanged:
			f.copy(mac, l, fr)
			first()
		}
	}
	if ctx.Err() == nil {
		f.lost(mac, l, why)
	}
}

func (f *Far) copy(mac string, l *link, fr engine.StateFrame) {
	// invariant: a snapshot is the whole of what the other Mac exports: a
	// key it no longer names is dropped; a change names its keys gone.
	f.mu.Lock()
	if self := selfOf(maps.Keys(fr.Slices)); self != "" {
		l.self = self
	}
	copied := renamed(mac, l.self, fr.Slices)
	gone := slices.Collect(maps.Keys(l.keys))
	if fr.Event == engine.StateChanged {
		gone = nil
	}
	for _, k := range fr.Gone {
		gone = append(gone, rename(mac, l.self, store.Key(k)))
	}
	for k := range copied {
		l.keys[k] = true
		gone = slices.DeleteFunc(gone, func(g store.Key) bool { return g == k })
	}
	for _, k := range gone {
		delete(l.keys, k)
	}
	l.fails, l.retry = 0, time.Time{}
	delete(f.held, mac)
	if l.since != nil {
		l.since.Rev = fr.Rev
	}
	f.mu.Unlock()
	for _, k := range slices.Sorted(maps.Keys(copied)) {
		f.apply(k, copied[k])
	}
	for _, k := range gone {
		f.Store.Drop(k)
	}
}

func (f *Far) apply(k store.Key, sl engine.SliceOut) {
	if _, err := store.ParseKey(string(k)); err != nil {
		return
	}
	readAt, _ := time.Parse(time.RFC3339, sl.ReadAt)
	if sl.Value != nil {
		v, err := decode(k.Family(), sl.Value)
		if err != nil {
			_ = f.Store.Fail(k, "unreadable")
			return
		}
		_ = f.Store.Put(store.Put{Key: k, Value: v, ReadAt: readAt})
	}
	switch {
	case sl.State == string(store.Fresh) && sl.Value == nil:
		if held, ok := f.Store.Now().Slices[k]; ok && held.State != store.Fresh && held.Value != nil {
			_ = f.Store.Put(store.Put{Key: k, Value: held.Value, ReadAt: readAt})
		}
	case sl.State != string(store.Fresh):
		_ = f.Store.Fail(k, sl.Why)
	}
}

func (f *Far) lost(mac string, l *link, why string) {
	// invariant: a Mac that stops pushing goes stale: every slice copied of
	// it keeps its last value, marked stale with why; one never heard is
	// unread with why, its own slice at least.
	f.mu.Lock()
	if !l.running {
		f.mu.Unlock()
		return
	}
	l.cancel()
	l.running = false
	l.retry = f.Clock.Now().Add(farWait(l.fails))
	l.fails++
	held := renamed(mac, selfOf(maps.Keys(f.held[mac])), f.held[mac])
	delete(f.held, mac)
	for k := range held {
		l.keys[k] = true
	}
	keys := slices.Collect(maps.Keys(l.keys))
	f.mu.Unlock()
	for _, k := range slices.Sorted(maps.Keys(held)) {
		f.apply(k, held[k])
	}
	if own := store.Of(store.FamilyMac, mac); !slices.Contains(keys, own) {
		keys = append(keys, own)
	}
	slices.Sort(keys)
	for _, k := range keys {
		_ = f.Store.Fail(k, why)
	}
	if f.Drop != nil {
		f.Drop(mac)
	}
}

func selfOf(keys iter.Seq[string]) string {
	for k := range keys {
		if key := store.Key(k); key.Family() == store.FamilyMac {
			return key.Machine()
		}
	}
	return ""
}

func renamed(mac, self string, in map[string]engine.SliceOut) map[store.Key]engine.SliceOut {
	out := make(map[store.Key]engine.SliceOut, len(in))
	for k, sl := range in {
		if self != "" && self != mac && store.Key(k).Family().PerMachine() {
			sl.Value = relabel(sl.Value, self, mac)
		}
		out[rename(mac, self, store.Key(k))] = sl
	}
	return out
}

// invariant: in a per-machine slice another Mac exports, every value that
// is its engine's own name is the name This Mac added it by, wherever it
// sits; a sentence its engine wrote, and a tailnet's or a record's slice,
// are held as they came.
func relabel(raw json.RawMessage, self, mac string) json.RawMessage {
	if raw == nil {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if dec.Decode(&v) != nil {
		return raw
	}
	out, err := json.Marshal(swap(v, self, mac))
	if err != nil {
		return raw
	}
	return out
}

func swap(v any, self, mac string) any {
	switch t := v.(type) {
	case string:
		if t == self {
			return mac
		}
	case []any:
		for i := range t {
			t[i] = swap(t[i], self, mac)
		}
	case map[string]any:
		for k := range t {
			t[k] = swap(t[k], self, mac)
		}
	}
	return v
}

func rename(mac, self string, k store.Key) store.Key {
	if self == "" || self == mac || !k.Family().PerMachine() || k.Machine() != self {
		return k
	}
	return store.Of(k.Family(), mac)
}

func (f *Far) watchdog(ctx context.Context, mac string, l *link) {
	for f.Clock.Sleep(ctx, farWatch) == nil {
		f.mu.Lock()
		silent := f.Clock.Now().Sub(l.heard) >= farSilent
		f.mu.Unlock()
		if silent {
			f.lost(mac, l, store.WhyUnreachable)
			return
		}
	}
}

func stamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func decode(family store.Family, raw json.RawMessage) (any, error) {
	// invariant: a copied slice is held as the Go value its family's reader
	// puts, so a projection reads another Mac's slice as it reads This Mac's.
	switch family {
	case store.FamilyMac:
		return decodeAs[macView](raw)
	case store.FamilyPorts:
		return decodeAs[macPorts](raw)
	case store.FamilyOverview:
		return decodeAs[overviewKept](raw)
	case store.FamilyAwake:
		return decodeAs[awakeReport](raw)
	case store.FamilyActivity:
		return decodeAs[activityReport](raw)
	case store.FamilySignins:
		return decodeAs[signinsRead](raw)
	case store.FamilyTailnet, store.FamilyRecords, store.FamilyVault, store.FamilyNode:
		return decodeAs[any](raw)
	}
	return nil, fmt.Errorf("no family %q", family)
}

func decodeAs[T any](raw json.RawMessage) (any, error) {
	var v T
	err := json.Unmarshal(raw, &v)
	return v, err
}
