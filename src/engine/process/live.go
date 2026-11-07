package process

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/jterrazz/studio-station/src/engine"
)

type liveRun struct {
	n    int
	stop context.CancelFunc
}

// invariant: one reading of the live keys serves every watch of pages: while
// a watch is open the engine reads, every 2 s, the keys its open pages list,
// all watches' at once, and hands each watch the values of its own pages'
// keys, so two pages that show one key show one value in one tick; once the
// last watch ends nothing is read.
func (d *Engine) joinLive() func() {
	if d.o.Sample == nil {
		return func() {}
	}
	d.hub.mu.Lock()
	defer d.hub.mu.Unlock()
	d.hub.live.n++
	if d.hub.live.n == 1 {
		ctx, stop := context.WithCancel(context.Background())
		d.hub.live.stop = stop
		go d.sampleLive(ctx)
	}
	return func() {
		d.hub.mu.Lock()
		defer d.hub.mu.Unlock()
		d.hub.live.n--
		if d.hub.live.n == 0 {
			d.hub.live.stop()
		}
	}
}

func (d *Engine) sampleLive(ctx context.Context) {
	for d.h.Sleep(ctx, engine.SampleEvery) == nil {
		keys := d.hub.liveKeys()
		if len(keys) == 0 {
			continue
		}
		s := d.o.Sample(ctx, keys)
		if ctx.Err() != nil {
			return
		}
		d.hub.sampled(d.h.Now().UTC().Format(time.RFC3339), s)
	}
}

// value: the live keys a page lists, read from its frame as it streams.
func liveOf(data json.RawMessage) []string {
	var p struct {
		Live []string `json:"live"`
	}
	_ = json.Unmarshal(data, &p)
	return p.Live
}

func (h *hub) listLive(s *subscriber, page string, keys []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	s.live[page] = keys
}

func (h *hub) liveKeys() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	var keys []string
	for s := range h.subs {
		keys = append(keys, s.wanted()...)
	}
	return slices.Compact(slices.Sorted(slices.Values(keys)))
}

func (s *subscriber) wanted() []string {
	var keys []string
	for _, k := range s.live {
		keys = append(keys, k...)
	}
	return slices.Compact(slices.Sorted(slices.Values(keys)))
}

// invariant: a watch is handed only its own pages' keys, the latest tick
// replacing one it has not read yet, so a slow reader never blocks the tick.
func (h *hub) sampled(at string, sample engine.Sample) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for s := range h.subs {
		e := engine.Event{Event: engine.EventSample, At: at, Values: map[string]any{}}
		for _, k := range s.wanted() {
			if v, ok := sample.Values[k]; ok {
				e.Values[k] = v
			}
			if slices.Contains(sample.Stale, k) {
				e.Stale = append(e.Stale, k)
			}
		}
		if len(e.Values) == 0 {
			continue
		}
		select {
		case <-s.sample:
		default:
		}
		s.sample <- e
	}
}
