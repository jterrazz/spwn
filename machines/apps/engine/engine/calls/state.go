package calls

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/store"
)

const (
	StateNoKey      fail.Code = "engine.state-no-key"
	StateUnknownKey fail.Code = "engine.state-unknown-key"
	// value: how long the changes of a held stream are gathered into one
	// frame, and the period of its tick, its liveness.
	stateCoalesce = 250 * time.Millisecond
	stateTick     = 15 * time.Second
)

type State interface {
	Store() *store.Store
	// value: whether a key is one this Mac reads itself, the keys `local`
	// names.
	Owns(k store.Key) bool
	// value: holds the readers at their attached pace until released.
	Attach() (release func())
	// value: the store holds what the engine reads before it answers.
	Ready(ctx context.Context)
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

type Views interface {
	// value: the window's views: each a projection of one snapshot, named
	// as the call whose answer it is.
	Known(name string) bool
	Project(s *store.Snapshot, asked func(name string) bool) map[string]store.View
}

type stateKey struct{}

func WithState(ctx context.Context, s State) context.Context {
	// value: the store of the engine that serves a connection, which its
	// engine.state answers from.
	return context.WithValue(ctx, stateKey{}, s)
}

func WithoutState(ctx context.Context) context.Context {
	// value: a context that carries no engine's store, so an engine.state
	// asked within it is answered by the engine it reaches.
	return context.WithValue(ctx, stateKey{}, nil)
}

var StateWatch = call.Watch[Deps, engine.StateIn, engine.StateFrame]{
	Name: "engine.state", Where: call.InEngine, Applies: Applies,
	Errors: []fail.Code{StateNoKey, StateUnknownKey},
	Run: func(ctx context.Context, d Deps, in engine.StateIn, _ call.Progress) (<-chan call.Update[engine.StateFrame], error) {
		// invariant: ADR-070: the opening carries the verdict, then a
		// fitting stream answers at once from what the store holds, never
		// from a read: a snapshot, or the keys changed since the revision
		// the client names; held, a change per gathered revision and a tick
		// every 15 s.
		patterns, err := patternsOf(in.Keys, d.Views)
		if err != nil {
			return nil, err
		}
		src, ok := ctx.Value(stateKey{}).(State)
		if !ok {
			return forward(ctx, d, in)
		}
		out := make(chan call.Update[engine.StateFrame], 1)
		go stream(ctx, src, d.Page, patterns, in, out)
		return out, nil
	},
}

// invariant: a door that is not the engine holds no store: its stream is
// the engine's, relayed frame for frame.
func forward(ctx context.Context, d Deps, in engine.StateIn) (<-chan call.Update[engine.StateFrame], error) {
	if d.Forward == nil {
		return nil, &DownError{}
	}
	frames, err := d.Forward(ctx, in)
	if err != nil {
		return nil, err
	}
	out := make(chan call.Update[engine.StateFrame])
	go func() {
		defer close(out)
		for raw := range frames {
			var f engine.StateFrame
			if err := json.Unmarshal(raw, &f); err != nil {
				out <- call.Update[engine.StateFrame]{Err: err}
				return
			}
			select {
			case out <- call.Update[engine.StateFrame]{Value: f}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

// value: what a stream is asked: the slices by their patterns, and the
// views by their names.
type asked struct {
	slices []store.Pattern
	views  []string
	of     Views
}

func patternsOf(keys []string, views Views) (asked, error) {
	if len(keys) == 0 {
		return asked{}, &fail.Error{Type: fail.Refused, Code: StateNoKey, Field: fail.FieldKeys}
	}
	out := asked{of: views}
	for _, k := range keys {
		if views != nil && views.Known(k) {
			out.views = append(out.views, k)
			continue
		}
		p, err := store.ParsePattern(k)
		if err != nil {
			return asked{}, &fail.Error{Type: fail.Refused, Code: StateUnknownKey, Field: fail.FieldKeys, Subject: k, Cause: err}
		}
		out.slices = append(out.slices, p)
	}
	return out, nil
}

func (a asked) view(name string) bool {
	// invariant: a view is asked by its name, or with every view of its kind
	// as `<kind>:*`.
	for _, v := range a.views {
		if v == name {
			return true
		}
		if kind, ok := strings.CutSuffix(v, ":*"); ok && strings.HasPrefix(name, kind+":") {
			return true
		}
	}
	return false
}

// value: the asked views of a snapshot, encoded as a slice is.
func (a asked) project(st *store.Store, s *store.Snapshot) map[string]viewed {
	if a.of == nil || len(a.views) == 0 {
		return nil
	}
	out := map[string]viewed{}
	for name, v := range a.of.Project(s, a.view) {
		raw, hash, err := st.Encode(v.Value)
		if err != nil {
			continue
		}
		out[name] = viewed{View: v, raw: raw, hash: hash}
	}
	return out
}

type viewed struct {
	store.View
	raw  json.RawMessage
	hash string
}

func (v viewed) out(value bool) engine.SliceOut {
	o := engine.SliceOut{Hash: v.hash, ReadAt: stamp(v.ReadAt), State: string(v.State), Why: v.Why}
	if value {
		o.Value = v.raw
	}
	return o
}

func stream(ctx context.Context, src State, page int, patterns asked, in engine.StateIn, out chan<- call.Update[engine.StateFrame]) {
	defer close(out)
	send := func(f engine.StateFrame) bool {
		select {
		case out <- call.Update[engine.StateFrame]{Value: f}:
			return true
		case <-ctx.Done():
			return false
		}
	}
	src.Ready(ctx)
	st := src.Store()
	now := st.Now()
	fit := engine.FitOf(engine.StateRevision, in.Revision)
	opened := engine.StateFrame{Event: engine.StateOpened, Stream: engine.StateStream, State: engine.StateRevision, Page: page, Epoch: string(now.Epoch), Fit: &fit, Rev: now.Rev, At: stamp(now.At)}
	if !send(opened) || fit.Verdict != engine.Fits {
		return
	}
	if in.Hold && in.Pace != engine.PaceIdle {
		defer src.Attach()()
	}
	watch := st.Watch(ctx)
	now = <-watch
	matches := func(k store.Key) bool {
		return slices.ContainsFunc(patterns.slices, func(p store.Pattern) bool {
			return p.Match(k, func(string) bool { return src.Owns(k) })
		})
	}
	first := engine.StateFrame{Event: engine.StateSnapshot, Rev: now.Rev, At: stamp(now.At), Slices: map[string]engine.SliceOut{}}
	if in.Since != nil {
		if changed, gone, ok := st.Since(store.Epoch(in.Since.Epoch), in.Since.Rev); ok {
			first.Event = engine.StateChanged
			for _, k := range changed {
				if sl, held := now.Slices[k]; held && matches(k) {
					first.Slices[string(k)] = sliceOut(sl, true)
				}
			}
			for _, k := range gone {
				if matches(k) {
					first.Gone = append(first.Gone, string(k))
				}
			}
		}
	}
	if first.Event == engine.StateSnapshot {
		for k, sl := range now.Slices {
			if matches(k) {
				first.Slices[string(k)] = sliceOut(sl, true)
			}
		}
	}
	// invariant: a view is sent whole on a resume too: the client's revision
	// says which slices it holds, never which projection of them.
	views := patterns.project(st, now)
	for name, v := range views {
		first.Slices[name] = v.out(true)
	}
	if !send(first) || !in.Hold {
		return
	}
	clock := src
	ticks := make(chan struct{}, 1)
	go func() {
		for clock.Sleep(ctx, stateTick) == nil {
			select {
			case ticks <- struct{}{}:
			default:
			}
		}
	}()
	last := now
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			n := st.Now()
			if !send(engine.StateFrame{Event: engine.StateTick, Rev: n.Rev, At: stamp(n.At)}) {
				return
			}
		case next, ok := <-watch:
			if !ok || clock.Sleep(ctx, stateCoalesce) != nil {
				return
			}
			select {
			case later, ok := <-watch:
				if !ok {
					return
				}
				next = later
			default:
			}
			f, _ := changedSince(last, next, matches)
			projected := patterns.project(st, next)
			f = viewsChanged(f, views, projected)
			if (len(f.Slices) > 0 || len(f.Gone) > 0) && !send(f) {
				return
			}
			last, views = next, projected
		}
	}
}

func changedSince(last, next *store.Snapshot, matches func(store.Key) bool) (engine.StateFrame, bool) {
	f := engine.StateFrame{Event: engine.StateChanged, Rev: next.Rev, At: stamp(next.At), Slices: map[string]engine.SliceOut{}}
	for k, sl := range next.Slices {
		if !matches(k) {
			continue
		}
		was, held := last.Slices[k]
		if held && was.Rev == sl.Rev {
			continue
		}
		f.Slices[string(k)] = sliceOut(sl, !held || was.Hash != sl.Hash)
	}
	for k := range last.Slices {
		if _, held := next.Slices[k]; !held && matches(k) {
			f.Gone = append(f.Gone, string(k))
		}
	}
	slices.Sort(f.Gone)
	return f, len(f.Slices) > 0 || len(f.Gone) > 0
}

// invariant: a view moves when its bytes or its state do; one whose bytes
// did not move is sent without its value.
func viewsChanged(f engine.StateFrame, was, now map[string]viewed) engine.StateFrame {
	for name, v := range now {
		old, held := was[name]
		if held && old.hash == v.hash && old.State == v.State && old.Why == v.Why {
			continue
		}
		f.Slices[name] = v.out(!held || old.hash != v.hash)
	}
	for name := range was {
		if _, held := now[name]; !held {
			f.Gone = append(f.Gone, name)
		}
	}
	slices.Sort(f.Gone)
	return f
}

func sliceOut(sl store.Slice, value bool) engine.SliceOut {
	o := engine.SliceOut{Hash: sl.Hash, ReadAt: stamp(sl.ReadAt), State: string(sl.State), Why: sl.Why}
	if value {
		o.Value = sl.Raw
	}
	return o
}

func stamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
