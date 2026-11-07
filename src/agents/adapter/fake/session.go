package fake

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/src/agents/contract"
	"github.com/jterrazz/studio-station/src/world"
)

type Session struct {
	// invariant: the agents' terminal session of one VM of the world.
	inVM
}

var _ contract.Session = (*Session)(nil)

func NewSession() *Session { return &Session{alone()} }

func SessionOf(w *world.World, vm string) *Session { return &Session{inVM{w: w, vm: vm}} }

func (s *Session) Open(_ context.Context, folder string, argv []string) error {
	s.w.Lock()
	defer s.w.Unlock()
	held := s.agents()
	held.Opened = append(held.Opened, strings.TrimSpace(folder+" "+strings.Join(argv, " ")))
	return nil
}

func (s *Session) Send(_ context.Context, text string) error {
	s.w.Lock()
	defer s.w.Unlock()
	held := s.agents()
	if len(held.Opened) == 0 {
		return errors.New("no agent runs in the session")
	}
	held.Typed = append(held.Typed, text)
	return nil
}

func (s *Session) Opened() []string {
	s.w.Lock()
	defer s.w.Unlock()
	return slices.Clone(s.agents().Opened)
}

func (s *Session) Typed() []string {
	s.w.Lock()
	defer s.w.Unlock()
	return slices.Clone(s.agents().Typed)
}
