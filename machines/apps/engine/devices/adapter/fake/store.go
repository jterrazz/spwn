package fake

import (
	"context"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/devices"
	"github.com/jterrazz/studio-station/apps/engine/devices/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Store struct {
	// invariant: one Mac's device book, in its slice of the world.
	w   *world.World
	mac string
}

var _ contract.Store = (*Store)(nil)

func StoreOf(w *world.World, mac string) *Store { return &Store{w: w, mac: mac} }

func clone(in []devices.Device) []devices.Device {
	out := make([]devices.Device, 0, len(in))
	for _, d := range in {
		d.Channels = slices.Clone(d.Channels)
		d.Seal = slices.Clone(d.Seal)
		out = append(out, d)
	}
	return out
}

func (s *Store) Read(context.Context) (devices.Book, error) {
	s.w.Lock()
	defer s.w.Unlock()
	return devices.Book{Devices: clone(s.w.Mac(s.mac).Store().Devices)}, nil
}

func (s *Store) Change(_ context.Context, edit func(*devices.Book) error) error {
	s.w.Lock()
	defer s.w.Unlock()
	st := s.w.Mac(s.mac).Store()
	next := devices.Book{Devices: clone(st.Devices)}
	if err := edit(&next); err != nil {
		return err
	}
	st.Devices = clone(next.Devices)
	return nil
}
