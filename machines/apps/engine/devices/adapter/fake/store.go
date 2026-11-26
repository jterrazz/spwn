package fake

import (
	"context"
	"errors"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/devices"
	"github.com/jterrazz/studio-station/apps/engine/devices/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Store struct {
	// invariant: one Mac's device book, in its slice of the world, its
	// head anchored in the same slice as its login keychain keeps it.
	w   *world.World
	mac string
	// value: this process is its Mac's engine, from its start read on.
	sole bool
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
// entry, which its head covers; a file that does not parse is no book's.
func sumOf(st *world.Stores) string {
	if st.DevicesUnreadable {
		return "unreadable"
	}
	return bookOf(st).Head().String()
}

// value: where the book's file sits, as a refusal to read it names it.
const bookPath = "~/.local/state/station/devices.json"

// outside: what encoding/json says of a file that is not JSON.
var errSyntax = errors.New("invalid character 'g' looking for beginning of value")

// invariant: a book whose file does not parse is read by this Mac's running
// engine, past its wait, as an empty book under the head it holds; by any
// other reader, and while the engine awaits, it is refused.
func unreadable(m *world.Mac, sole bool) (devices.Head, error) {
	if sole && m.EngineRuns() && m.Engine.Book != nil && m.Engine.Book.State != devices.Awaiting {
		return m.Engine.Book.Head, nil
	}
	return devices.Head{}, &store.UnreadableError{Path: bookPath, Cause: errSyntax, Line: 1}
}

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
	return read(s.w.Mac(s.mac), s.sole)
}

func read(m *world.Mac, sole bool) (devices.Book, devices.Head, error) {
	st := m.Store()
	if st.DevicesUnreadable {
		h, err := unreadable(m, sole)
		return devices.Book{Devices: []devices.Device{}}, h, err
	}
	h, err := held(m)
	if err != nil {
		return devices.Book{}, devices.Head{}, err
	}
	return bookOf(st), h, nil
}

func (s *Store) Change(_ context.Context, edit func(*devices.Book) error) error {
	s.w.Lock()
	defer s.w.Unlock()
	m := s.w.Mac(s.mac)
	book, h, err := read(m, s.sole)
	if err != nil {
		return err
	}
	next, err := devices.Next(book, h, edit)
	if err != nil {
		return err
	}
	st := m.Store()
	head := next.Head()
	st.DevicesAnchor = head.String()
	st.Devices, st.DevicesGeneration, st.DevicesUnreadable = clone(next.Devices), next.Generation, false
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
	s.sole = true
	m := s.w.Mac(s.mac)
	if !m.EngineRuns() || m.Engine.Book != nil {
		return
	}
	st := m.Store()
	got, err := anchored(st)
	hold := devices.Keep(got)
	switch {
	case st.DevicesUnreadable:
		hold = devices.Refuse(0)
	case err == nil:
	case malformed(st, err):
		hold = devices.Refuse(st.DevicesGeneration)
	default:
		hold = devices.Await(bookOf(st), sumOf(st))
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
	st := m.Store()
	found := devices.ABook
	if st.DevicesUnreadable {
		found = devices.NotABook
	}
	return m.Engine.Book.Head, m.Engine.Book.Standing(bookOf(st), found), true
}
