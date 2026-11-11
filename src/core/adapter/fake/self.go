package fake

import (
	"context"

	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/world"
)

type Self struct {
	w   *world.World
	mac string
}

func NewSelf(p machine.Profile) *Self {
	// value: the machine a spec stands on, by its profile; its capabilities
	// are the other contracts', so Self drops them.
	w := &world.World{ThisMac: p.Ref.Name}
	w.Mac(p.Ref.Name).Self = &world.Self{Kind: p.Kind, System: p.System, Hardware: p.Hardware}
	return SelfOf(w, p.Ref.Name)
}

func SelfOf(w *world.World, mac string) *Self {
	// value: a Mac of the world as it reads itself; one whose slice says
	// nothing of itself is a Mac on macOS.
	return &Self{w: w, mac: mac}
}

func (s *Self) Read(context.Context) (machine.Ref, machine.Kind, machine.System, machine.Hardware, error) {
	s.w.Lock()
	defer s.w.Unlock()
	self := s.w.Mac(s.mac).Self
	if self == nil {
		self = &world.Self{}
	}
	kind, sys := self.Kind, self.System
	if kind == "" {
		kind = machine.Computer
	}
	if sys.OS == "" {
		sys.OS = machine.MacOS
	}
	return machine.Ref{Name: s.mac}, kind, sys, self.Hardware, nil
}
