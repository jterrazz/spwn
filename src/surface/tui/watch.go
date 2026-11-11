package tui

import (
	"slices"

	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/core/section"
)

type held struct {
	on      []string
	release func()
}

func (h *held) end() {
	if h != nil && h.release != nil {
		h.release()
		h.release, h.on = nil, nil
	}
}

// invariant: SPEC §6.3: while the TUI shows a machine's activity (its
// Activity, its Overview, or All machines for every one) it holds this
// Mac's engine's watch on it, so the engine samples it every 2 s; the watch
// follows the screen and ends with it. A machine of another Mac is that
// Mac's engine's to sample.
func (s shell) watched() []string {
	n := s.current()
	if id := s.section(); id != section.Overview && id != section.Activity {
		return nil
	}
	var out []string
	for _, x := range s.nodes[1:] {
		mine := x.thisMac || (x.kind != machine.Computer && x.mac == s.deps.ThisMachine.Machine)
		if mine && (n.all() || x.name == n.name) {
			out = append(out, x.name)
		}
	}
	return out
}

func (s shell) hold() {
	if s.deps.Watch == nil || s.held == nil {
		return
	}
	want := s.watched()
	if slices.Equal(want, s.held.on) {
		return
	}
	s.held.end()
	if len(want) > 0 {
		s.held.on, s.held.release = want, s.deps.Watch(want)
	}
}
