package process

import (
	"encoding/json"
	"slices"
	"sync"

	"github.com/jterrazz/studio-station/apps/engine/engine/store"
)

// value: how many pages the engine keeps projected, the least recently
// asked dropped first.
const projectedKept = 64

type projected struct {
	epoch store.Epoch
	rev   uint64
	raw   json.RawMessage
}

// invariant: a page is projected once per revision of the store, under one
// lock, and its bytes shared by every watch of it.
type memo struct {
	mu    sync.Mutex
	of    map[string]projected
	order []string
}

func (d *Engine) projected(id string) bool {
	name, _ := feedOf(id)
	_, ok := d.o.Pages[name]
	return ok
}

func (d *Engine) project(id string, s *store.Snapshot) (json.RawMessage, bool) {
	m := &d.memo
	m.mu.Lock()
	defer m.mu.Unlock()
	m.order = append(slices.DeleteFunc(m.order, func(o string) bool { return o == id }), id)
	if p, ok := m.of[id]; ok && p.epoch == s.Epoch && p.rev == s.Rev {
		return p.raw, true
	}
	name, subject := feedOf(id)
	raw, err := json.Marshal(d.o.Pages[name](s, subject))
	if err != nil {
		return nil, false
	}
	if m.of == nil {
		m.of = map[string]projected{}
	}
	m.of[id] = projected{epoch: s.Epoch, rev: s.Rev, raw: raw}
	for len(m.order) > projectedKept {
		delete(m.of, m.order[0])
		m.order = m.order[1:]
	}
	return raw, true
}
