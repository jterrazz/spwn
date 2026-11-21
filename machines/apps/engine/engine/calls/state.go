package calls

import (
	"context"
	"encoding/json"
	"slices"
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
		patterns, err := patternsOf(in.Keys)
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

func patternsOf(keys []string) ([]store.Pattern, error) {
	if len(keys) == 0 {
		return nil, &fail.Error{Type: fail.Refused, Code: StateNoKey, Field: fail.FieldKeys}
	}
	out := make([]store.Pattern, 0, len(keys))
	for _, k := range keys {
		p, err := store.ParsePattern(k)
		if err != nil {
			return nil, &fail.Error{Type: fail.Refused, Code: StateUnknownKey, Field: fail.FieldKeys, Subject: k, Cause: err}
		}
		out = append(out, p)
	}
	return out, nil
}

func stream(ctx context.Context, src State, page int, patterns []store.Pattern, in engine.StateIn, out chan<- call.Update[engine.StateFrame]) {
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
		return slices.ContainsFunc(patterns, func(p store.Pattern) bool {
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
			if f, moved := changedSince(last, next, matches); moved && !send(f) {
				return
			}
			last = next
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
