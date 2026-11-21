package process

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
	"github.com/jterrazz/studio-station/apps/engine/engine/store"
)

const (
	coalesce     = 250 * time.Millisecond
	tickEvery    = 15 * time.Second
	streamSchema = "station.page/v1"
)

type Projection func(s *store.Snapshot, subject string) any

type LivePage func(ctx context.Context, subject string) <-chan any

// value: a watched page's name and its subject, `<page>:<subject>`.
func feedOf(id string) (feed, subject string) {
	feed, subject, _ = strings.Cut(id, ":")
	return feed, subject
}

type subscriber struct {
	change chan struct{}
	notice chan engine.Notice
	// value: the live keys each of its pages lists, and the latest tick of
	// their values not yet written.
	live   map[string][]string
	sample chan engine.Event
}

type hub struct {
	mu   sync.Mutex
	subs map[*subscriber]bool
	seen string
	live liveRun
}

func (h *hub) add() *subscriber {
	s := &subscriber{change: make(chan struct{}, 1), notice: make(chan engine.Notice, 8), live: map[string][]string{}, sample: make(chan engine.Event, 1)}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subs == nil {
		h.subs = map[*subscriber]bool{}
	}
	h.subs[s] = true
	return s
}

func (h *hub) remove(s *subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.subs, s)
}

func signal(c chan struct{}) {
	select {
	case c <- struct{}{}:
	default:
	}
}

// invariant: a subscriber that does not read misses signals, never blocks the
// daemon: every send is a non-blocking one into a small buffer.
func (h *hub) changed() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for s := range h.subs {
		signal(s.change)
	}
}

func (h *hub) tell(n engine.Notice) {
	if n.Icon == "" {
		n.Icon = engine.NoticeIcon(n.Kind)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for s := range h.subs {
		select {
		case s.notice <- n:
		default:
		}
	}
}

func (h *hub) listening() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs) > 0
}

// invariant: the daemon's own state is compared each tick while a client
// subscribes or readers keep the store, so a VM, guest or reservation change
// reaches the stream and the store within a tick; with neither nothing is
// read.
func (d *Engine) watchState() {
	if !d.hub.listening() && d.o.Reading == nil {
		return
	}
	now := string(mustJSON([]any{d.vmViews(), d.macPorts("")}))
	d.hub.mu.Lock()
	moved := d.hub.seen != now
	d.hub.seen = now
	d.hub.mu.Unlock()
	if moved {
		d.hub.changed()
		if d.o.Reading != nil {
			d.o.Reading.Moved()
		}
	}
}

func (d *Engine) known(id string) bool {
	if id == engine.PageKeepAwake {
		return d.o.KeepAwakePage != nil
	}
	name, _ := feedOf(id)
	_, projected := d.o.Pages[name]
	_, live := d.o.Live[name]
	return projected || live
}

func (d *Engine) CheckPages(pages []string) error {
	hint := "watch page with the pages menubar, menubar.machine:<name>, menubar.ports[:<mac>], menubar.awake, menubar.activity[:<window>], menubar.accounts[:<machine>], menubar.needs[:<machine>], menubar.find or keep-awake"
	if len(pages) == 0 {
		return fault(enginecalls.UnknownPage, "the watch names no page", hint)
	}
	for _, id := range pages {
		if !d.known(id) {
			return fault(enginecalls.UnknownPage, "unknown page \""+id+"\"", hint)
		}
	}
	return nil
}

type latest struct {
	mu    sync.Mutex
	pages map[string]json.RawMessage
}

func (l *latest) set(id string, p json.RawMessage) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pages[id] = p
}

func (l *latest) get(id string) (json.RawMessage, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	p, ok := l.pages[id]
	return p, ok
}

// value: a live page's readings, the latest kept for the stream, its first
// handed on first, closed if the page ends before any.
func (d *Engine) follow(ctx context.Context, id string, sub *subscriber, into *latest, first chan<- json.RawMessage) {
	name, subject := feedOf(id)
	framed := false
	for p := range d.o.Live[name](ctx, subject) {
		data, err := json.Marshal(p)
		if err != nil {
			continue
		}
		if !framed {
			framed = true
			first <- data
			continue
		}
		into.set(id, data)
		signal(sub.change)
	}
	if !framed {
		close(first)
	}
}

func (d *Engine) Stream(ctx context.Context, in engine.PageWatchIn, write func(any) error) error {
	// invariant: after its opening, a watch of pages only streams: its pages
	// at once from what the store holds (ADR-070), never after a read, then
	// the values the live ticks kept of their keys; then a full page per
	// change, coalesced over 250 ms (so at most 4 a second) and never resent
	// unchanged, the values of its pages' live keys every 2 s, a tick every
	// 15 s, and the notices. A watch whose verdict does not fit ends at its
	// opening: no page is ever sent to a client that cannot draw it.
	if err := d.CheckPages(in.Pages); err != nil {
		return err
	}
	pages := slices.Compact(slices.Sorted(slices.Values(in.Pages)))
	fit := engine.FitOf(d.o.Page, in.Revision)
	if fit.Verdict != engine.Fits {
		return write(engine.Opened{Stream: streamSchema, Pages: pages, Page: d.o.Page, Fit: &fit})
	}
	sub := d.hub.add()
	defer d.hub.remove(sub)
	defer d.joinLive(ctx)()
	if err := write(engine.Opened{Stream: streamSchema, Pages: pages, Page: d.o.Page, Fit: &fit}); err != nil {
		return err
	}
	opening := d.opening(pages)
	defer opening.end()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var snaps <-chan *store.Snapshot
	var snap *store.Snapshot
	if d.o.Reading != nil && slices.ContainsFunc(pages, d.projected) {
		d.o.Reading.Ready(ctx)
		snaps = d.o.Reading.Store().Watch(ctx)
		snap = <-snaps
	}
	fed := &latest{pages: map[string]json.RawMessage{}}
	frames := map[string]chan json.RawMessage{}
	var follows sync.WaitGroup
	for _, id := range pages {
		if name, _ := feedOf(id); d.o.Live[name] != nil {
			frame := make(chan json.RawMessage, 1)
			frames[id] = frame
			follows.Add(1)
			go func() {
				defer follows.Done()
				d.follow(ctx, id, sub, fed, frame)
			}()
		}
	}
	defer follows.Wait()
	defer cancel()
	for id, frame := range frames {
		select {
		case p, ok := <-frame:
			if ok {
				fed.set(id, p)
			}
		case <-ctx.Done():
			return nil
		}
	}

	ticks := make(chan struct{}, 1)
	go func() {
		for d.h.Sleep(ctx, tickEvery) == nil {
			signal(ticks)
		}
	}()
	last := map[string]string{}
	pageOf := func(id string) (json.RawMessage, bool) {
		switch {
		case id == engine.PageKeepAwake:
			data, err := json.Marshal(d.o.KeepAwakePage(d.awakeJob.View()))
			return data, err == nil
		case d.projected(id) && snap != nil:
			return d.project(id, snap)
		}
		return fed.get(id)
	}
	send := func() bool {
		for _, id := range pages {
			p, ok := pageOf(id)
			if !ok || last[id] == string(p) {
				continue
			}
			last[id] = string(p)
			d.hub.listLive(sub, id, liveOf(p))
			if write(engine.Event{Event: "page", Page: p}) != nil {
				return false
			}
			opening.wrote(id, p)
		}
		return true
	}
	if !send() || !d.kept(sub, write) {
		return nil
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case next, ok := <-snaps:
			if !ok {
				return nil
			}
			snap = next
			if d.h.Sleep(ctx, coalesce) != nil {
				return nil
			}
			snap = newest(snaps, snap)
			if !send() {
				return nil
			}
		case <-sub.change:
			if d.h.Sleep(ctx, coalesce) != nil {
				return nil
			}
			select {
			case <-sub.change:
			default:
			}
			if !send() {
				return nil
			}
		case n := <-sub.notice:
			if write(engine.Event{Event: "notice", Notice: &n}) != nil {
				return nil
			}
		case e := <-sub.sample:
			if write(e) != nil {
				return nil
			}
		case <-ticks:
			if write(engine.Event{Event: "tick", At: d.h.Now().UTC().Format(time.RFC3339)}) != nil {
				return nil
			}
		}
	}
}

// value: the latest snapshot waiting, else the one held.
func newest(snaps <-chan *store.Snapshot, held *store.Snapshot) *store.Snapshot {
	select {
	case next, ok := <-snaps:
		if ok {
			return next
		}
	default:
	}
	return held
}

// invariant: a page's moving numbers follow it at once, as the live ticks
// kept them, so its first paint holds current values without a read.
func (d *Engine) kept(sub *subscriber, write func(any) error) bool {
	keys := sub.wanted()
	if d.o.KeptLive == nil || len(keys) == 0 {
		return true
	}
	s := d.o.KeptLive(keys)
	if len(s.Values) == 0 {
		return true
	}
	return write(engine.Event{Event: engine.EventSample, At: d.h.Now().UTC().Format(time.RFC3339), Values: s.Values, Stale: s.Stale}) == nil
}

func mustJSON(v any) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
