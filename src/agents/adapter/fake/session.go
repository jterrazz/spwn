package fake

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/agents/contract"
	"github.com/jterrazz/studio-station/src/world"
)

type Session struct {
	// invariant: the multiplexer of one VM of the world.
	inVM
}

var _ contract.Session = (*Session)(nil)

func NewSession() *Session { return &Session{alone()} }

func SessionOf(w *world.World, vm string) *Session { return &Session{inVM{w: w, vm: vm}} }

func (s *Session) Open(_ context.Context, app agents.App, folder string, argv []string) (string, error) {
	s.w.Lock()
	defer s.w.Unlock()
	return openOn(s.agents(), app, folder, argv), nil
}

func openOn(held *world.Agents, app agents.App, folder string, argv []string) string {
	// value: a terminal opened on a world VM's multiplexer, its address the
	// next one.
	address := "t" + strconv.Itoa(len(held.Terminals)+1)
	held.Opened = append(held.Opened, strings.TrimSpace(folder+" "+strings.Join(argv, " ")))
	held.Terminals = append(held.Terminals, world.Terminal{Terminal: agents.Terminal{Address: address, App: app, Folder: folder}})
	return address
}

func (s *Session) terminal(address string) (*world.Terminal, error) {
	held := s.agents()
	i := slices.IndexFunc(held.Terminals, func(t world.Terminal) bool { return t.Address == address })
	if i < 0 {
		return nil, errors.New("no terminal is at " + address)
	}
	return &held.Terminals[i], nil
}

func (s *Session) Send(_ context.Context, terminal, text string) error {
	s.w.Lock()
	defer s.w.Unlock()
	t, err := s.terminal(terminal)
	if err != nil {
		return err
	}
	t.Typed = append(t.Typed, text)
	held := s.agents()
	held.Typed = append(held.Typed, text)
	return nil
}

func (s *Session) Screen(_ context.Context, terminal string) (string, error) {
	s.w.Lock()
	defer s.w.Unlock()
	t, err := s.terminal(terminal)
	if err != nil {
		return "", err
	}
	return t.Screen, nil
}

func (s *Session) Terminals(context.Context) ([]agents.Terminal, error) {
	s.w.Lock()
	defer s.w.Unlock()
	out := make([]agents.Terminal, 0, len(s.agents().Terminals))
	for _, t := range s.agents().Terminals {
		out = append(out, t.Terminal)
	}
	return out, nil
}

func (s *Session) Opened() []string {
	s.w.Lock()
	defer s.w.Unlock()
	return slices.Clone(s.agents().Opened)
}

func (s *Session) Held() []world.Terminal {
	// value: the multiplexer's terminals now, each with what was typed into it.
	s.w.Lock()
	defer s.w.Unlock()
	return slices.Clone(s.agents().Terminals)
}

func (s *Session) Says(terminal string, seen agents.Seen, message, screen string) {
	s.w.Lock()
	defer s.w.Unlock()
	if t, err := s.terminal(terminal); err == nil {
		t.Seen, t.Message, t.Screen = seen, message, screen
	}
}
