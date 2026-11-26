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

// outside: what `security` says on a keychain it may not open.
const locked = "security: SecKeychainSearchCopyNext: User interaction is not allowed."

var errLocked = errors.New("the login keychain refused the device book's head")

func anchored(st *world.Stores) (devices.Head, error) {
	if st.DevicesAnchorUnreadable {
		return devices.Head{}, &contract.AnchorUnreadableError{Cause: locked}
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

// invariant: while its Mac's engine runs, the head it holds is the one
// every reader of the world takes, the anchor never read again: the head
// it read first, then the one it wrote last; with no engine running, the
// anchor's.
func held(m *world.Mac) (devices.Head, error) {
	st := m.Store()
	if !m.EngineRuns() {
		return anchored(st)
	}
	if m.Engine.Book != nil {
		return *m.Engine.Book, nil
	}
	h, err := anchored(st)
	if err == nil {
		m.Engine.Book = &h
	}
	return h, err
}

func (s *Store) Read(context.Context) (devices.Book, devices.Head, error) {
	s.w.Lock()
	defer s.w.Unlock()
	m := s.w.Mac(s.mac)
	h, err := held(m)
	if err != nil {
		return devices.Book{}, devices.Head{}, err
	}
	st := m.Store()
	return devices.Book{Generation: st.DevicesGeneration, Devices: clone(st.Devices)}, h, nil
}

func (s *Store) Change(_ context.Context, edit func(*devices.Book) error) error {
	s.w.Lock()
	defer s.w.Unlock()
	m := s.w.Mac(s.mac)
	h, err := held(m)
	if err != nil {
		return err
	}
	st := m.Store()
	next, err := devices.Next(devices.Book{Generation: st.DevicesGeneration, Devices: clone(st.Devices)}, h, edit)
	if err != nil {
		return err
	}
	head := next.Head()
	st.DevicesAnchor = head.String()
	st.Devices, st.DevicesGeneration = clone(next.Devices), next.Generation
	if m.EngineRuns() {
		m.Engine.Book = &head
	}
	return nil
}

func (s *Store) Sole() {}

func (s *Store) Held(context.Context) (devices.Head, bool, bool) {
	s.w.Lock()
	defer s.w.Unlock()
	m := s.w.Mac(s.mac)
	if !m.EngineRuns() || m.Engine.Book == nil {
		return devices.Head{}, false, false
	}
	st := m.Store()
	book := devices.Book{Generation: st.DevicesGeneration, Devices: st.Devices}
	return *m.Engine.Book, !book.Anchored(*m.Engine.Book), true
}
