package fake

import (
	"context"
	"errors"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/devices"
	"github.com/jterrazz/studio-station/apps/engine/devices/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Store struct {
	// invariant: one Mac's device book, in its slice of the world, its
	// head anchored in the same slice as its login keychain keeps it.
	w   *world.World
	mac string
}

var _ contract.Store = (*Store)(nil)

func StoreOf(w *world.World, mac string) *Store { return &Store{w: w, mac: mac} }

type Anchor struct {
	w   *world.World
	mac string
}

var _ contract.Anchor = Anchor{}

func AnchorOf(w *world.World, mac string) Anchor { return Anchor{w: w, mac: mac} }

var errLocked = errors.New("the login keychain is locked")

func anchored(st *world.Stores) (devices.Head, error) {
	if st.DevicesAnchorUnreadable {
		return devices.Head{}, errLocked
	}
	if st.DevicesAnchor == "" {
		return devices.Head{}, nil
	}
	return devices.ParseHead(st.DevicesAnchor)
}

func (a Anchor) Read(context.Context) (devices.Head, error) {
	a.w.Lock()
	defer a.w.Unlock()
	return anchored(a.w.Mac(a.mac).Store())
}

func (a Anchor) Write(_ context.Context, h devices.Head) error {
	a.w.Lock()
	defer a.w.Unlock()
	st := a.w.Mac(a.mac).Store()
	if st.DevicesAnchorUnreadable {
		return errLocked
	}
	st.DevicesAnchor = h.String()
	return nil
}

func clone(in []devices.Device) []devices.Device {
	out := make([]devices.Device, 0, len(in))
	for _, d := range in {
		d.Channels = slices.Clone(d.Channels)
		d.Seal = slices.Clone(d.Seal)
		out = append(out, d)
	}
	return out
}

func (s *Store) Read(context.Context) (devices.Book, devices.Head, error) {
	s.w.Lock()
	defer s.w.Unlock()
	st := s.w.Mac(s.mac).Store()
	h, err := anchored(st)
	if err != nil {
		return devices.Book{}, devices.Head{}, err
	}
	return devices.Book{Generation: st.DevicesGeneration, Devices: clone(st.Devices)}, h, nil
}

func (s *Store) Change(_ context.Context, edit func(*devices.Book) error) error {
	s.w.Lock()
	defer s.w.Unlock()
	st := s.w.Mac(s.mac).Store()
	h, err := anchored(st)
	if err != nil {
		return err
	}
	next, err := devices.Next(devices.Book{Generation: st.DevicesGeneration, Devices: clone(st.Devices)}, h, edit)
	if err != nil {
		return err
	}
	st.DevicesAnchor = next.Head().String()
	st.Devices, st.DevicesGeneration = clone(next.Devices), next.Generation
	return nil
}
