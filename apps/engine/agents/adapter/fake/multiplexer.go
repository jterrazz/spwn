package fake

import (
	"context"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/agents/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Multiplexer struct {
	// invariant: the multiplexer of one Mac of the world.
	w   *world.World
	mac string
}

var _ contract.Multiplexer = (*Multiplexer)(nil)

func NewMultiplexer() *Multiplexer {
	return &Multiplexer{w: &world.World{ThisMac: "this-mac"}, mac: "this-mac"}
}

func MultiplexerOf(w *world.World, mac string) *Multiplexer { return &Multiplexer{w: w, mac: mac} }

func (h *Multiplexer) Machines(context.Context) ([]string, error) {
	h.w.Lock()
	defer h.w.Unlock()
	return slices.Clone(h.w.Mac(h.mac).Reached), nil
}

func (h *Multiplexer) Add(_ context.Context, machine string) error {
	h.w.Lock()
	defer h.w.Unlock()
	m := h.w.Mac(h.mac)
	if !slices.Contains(m.Reached, machine) {
		m.Reached = append(m.Reached, machine)
	}
	return nil
}

func (h *Multiplexer) Remove(_ context.Context, machine string) error {
	h.w.Lock()
	defer h.w.Unlock()
	m := h.w.Mac(h.mac)
	m.Reached = slices.DeleteFunc(m.Reached, func(n string) bool { return n == machine })
	return nil
}

func (h *Multiplexer) Attach(_ context.Context, machine, terminal string) error {
	h.w.Lock()
	defer h.w.Unlock()
	m := h.w.Mac(h.mac)
	address := terminal
	if machine != "" {
		address = machine + "/" + terminal
	}
	m.Attached = append(m.Attached, address)
	return nil
}

func (h *Multiplexer) Reached() []string {
	h.w.Lock()
	defer h.w.Unlock()
	return slices.Clone(h.w.Mac(h.mac).Reached)
}

func (h *Multiplexer) Attached() []string {
	h.w.Lock()
	defer h.w.Unlock()
	return slices.Clone(h.w.Mac(h.mac).Attached)
}
