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

func bookOf(st *world.Stores) devices.Book {
	return devices.Book{Generation: st.DevicesGeneration, Devices: clone(st.Devices)}
}

// value: the world's book as bytes are to a file: its generation and every
// entry, which its head covers.
func sumOf(st *world.Stores) string { return bookOf(st).Head().String() }

func malformed(st *world.Stores, err error) bool {
	return err != nil && !st.DevicesAnchorUnreadable
}

// invariant: while its Mac's engine runs and has taken its start read, the
// head it holds is the one every reader of the world takes: the anchor is
// read again only while that hold awaits its first readable answer; with
// no engine running, or before its start read, the anchor's.
func held(m *world.Mac) (devices.Head, error) {
	st := m.Store()
	if !m.EngineRuns() || m.Engine.Book == nil {
		return anchored(st)
	}
	hold := *m.Engine.Book
	if hold.State == devices.Awaiting {
		got, err := anchored(st)
		switch {
		case err == nil:
			hold = hold.Settle(sumOf(st), got)
		case malformed(st, err):
			hold = devices.Refuse(hold.Head.Generation)
		default:
			return devices.Head{}, err
		}
		m.Engine.Book = &hold
	}
	return hold.Head, nil
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
		hold := devices.Keep(head)
		m.Engine.Book = &hold
	}
	return nil
}

func (s *Store) Sole(context.Context) {
	// invariant: the start read of the world's engine, taken once in its
	// lifetime and kept in its memory between commands; the next start
	// after a restart takes it again.
	s.w.Lock()
	defer s.w.Unlock()
	m := s.w.Mac(s.mac)
	if !m.EngineRuns() || m.Engine.Book != nil {
		return
	}
	st := m.Store()
	got, err := anchored(st)
	hold := devices.Keep(got)
	switch {
	case err == nil:
	case malformed(st, err):
		hold = devices.Refuse(st.DevicesGeneration)
	default:
		hold = devices.Await(st.DevicesGeneration, sumOf(st))
	}
	m.Engine.Book = &hold
}

func (s *Store) Held(context.Context) (devices.Head, devices.Standing, bool) {
	s.w.Lock()
	defer s.w.Unlock()
	m := s.w.Mac(s.mac)
	if !m.EngineRuns() || m.Engine.Book == nil {
		return devices.Head{}, devices.Kept, false
	}
	_, _ = held(m)
	return m.Engine.Book.Head, m.Engine.Book.Standing(bookOf(m.Store()), false), true
}
