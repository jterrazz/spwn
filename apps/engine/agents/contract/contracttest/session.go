package contracttest

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract"
)

type Typed struct{ Terminal, Text string }

type SessionSubject struct {
	Session contract.Session
	// value: the programs the multiplexer runs, each "<folder> <argv>", in the
	// order opened.
	Opened func() []string
	// value: the texts typed, each with the terminal it went to.
	Typed func() []Typed
	// value: every command line the adapter ran.
	Lines func() []string
	// value: makes the agent of a terminal say seen, with its words.
	Says func(terminal string, seen agents.Seen, message string)
}

func Session(t *testing.T, open func(t *testing.T) SessionSubject) {
	ctx := context.Background()

	t.Run("each program opened runs in its folder, in a terminal of its own", func(t *testing.T) {
		s := open(t)
		at := make([]string, 0, 2)
		for i, folder := range []string{"/Users/admin/app", "/Users/admin/api"} {
			address, err := s.Session.Open(ctx, "claudecode", []string{"linux-1", "linux-1" + agents.Nth + "2"}[i], folder, []string{"agent"})
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			at = append(at, address)
		}
		if got := s.Opened(); !slices.Equal(got, []string{"/Users/admin/app agent", "/Users/admin/api agent"}) {
			t.Errorf("opened %q, want both, in their folders", got)
		}
		if at[0] == "" || at[0] == at[1] || !agents.PlainSession(at[0]) || !agents.PlainSession(at[1]) {
			t.Errorf("addresses %q, want two plain ones apart", at)
		}
	})

	t.Run("a text sent is typed once with Enter into its terminal, and never on a command line", func(t *testing.T) {
		s := open(t)
		_, _ = s.Session.Open(ctx, "claudecode", "linux-1", "/Users/admin/app", []string{"agent"})
		address, err := s.Session.Open(ctx, "claudecode", "linux-1"+agents.Nth+"2", "/Users/admin/api", []string{"agent"})
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		text := "yes, merge it; then run `rm -rf build` && say \"done\" $HOME"
		if err := s.Session.Send(ctx, address, text); err != nil {
			t.Fatalf("Send: %v", err)
		}
		if got := s.Typed(); !slices.Equal(got, []Typed{{address, text}}) {
			t.Errorf("typed %q, want the text once into %s", got, address)
		}
		for _, line := range s.Lines() {
			if strings.Contains(line, "merge it") {
				t.Errorf("the text went on a command line: %q", line)
			}
		}
	})

	t.Run("the terminals opened are listed by name with what their agent says", func(t *testing.T) {
		s := open(t)
		address, err := s.Session.Open(ctx, "claudecode", "review-bot", "/Users/admin/app", []string{"agent"})
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		s.Says(address, agents.SeenBlocked, "Claude needs your permission to use Bash")
		live, err := s.Session.Terminals(ctx)
		if err != nil {
			t.Fatalf("Terminals: %v", err)
		}
		want := []agents.Terminal{{Address: address, Name: "review-bot", App: "claudecode", Folder: "/Users/admin/app", Seen: agents.SeenBlocked, Message: "Claude needs your permission to use Bash"}}
		if !slices.Equal(live, want) {
			t.Errorf("terminals %+v, want %+v", live, want)
		}
	})

	t.Run("a terminal closed is no longer listed, and closing it again changes nothing", func(t *testing.T) {
		s := open(t)
		gone, err := s.Session.Open(ctx, "claudecode", "linux-1", "/Users/admin/app", []string{"agent"})
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		for range 2 {
			if closeErr := s.Session.Close(ctx, gone); closeErr != nil {
				t.Fatalf("Close: %v", closeErr)
			}
		}
		live, err := s.Session.Terminals(ctx)
		if err != nil || slices.ContainsFunc(live, func(t agents.Terminal) bool { return t.Address == gone }) {
			t.Errorf("terminals %+v (%v), want %s gone", live, err, gone)
		}
	})
}

type MultiplexerSubject struct {
	Multiplexer contract.Multiplexer
	// value: the machines the multiplexer reaches now, by name.
	Reached func() []string
	// value: the terminals the owner's terminal was put on, each
	// "<machine>/<terminal>", or the terminal alone on this machine.
	Attached func() []string
}

func Multiplexer(t *testing.T, open func(t *testing.T) MultiplexerSubject) {
	ctx := context.Background()

	t.Run("a machine added is reached once, and forgotten when removed", func(t *testing.T) {
		h := open(t)
		for range 2 {
			if err := h.Multiplexer.Add(ctx, "linux-1"); err != nil {
				t.Fatalf("Add: %v", err)
			}
		}
		if got, err := h.Multiplexer.Machines(ctx); err != nil || !slices.Equal(got, []string{"linux-1"}) {
			t.Errorf("machines %q (%v), want linux-1 once", got, err)
		}
		for range 2 {
			if err := h.Multiplexer.Remove(ctx, "linux-1"); err != nil {
				t.Fatalf("Remove: %v", err)
			}
		}
		if got := h.Reached(); len(got) != 0 {
			t.Errorf("reached %q, want none", got)
		}
	})

	t.Run("attach puts the owner's terminal on a terminal of a machine, or of this one", func(t *testing.T) {
		h := open(t)
		if err := h.Multiplexer.Attach(ctx, "linux-1", "t7"); err != nil {
			t.Fatalf("Attach: %v", err)
		}
		if err := h.Multiplexer.Attach(ctx, "", "t2"); err != nil {
			t.Fatalf("Attach: %v", err)
		}
		if got := h.Attached(); !slices.Equal(got, []string{"linux-1/t7", "t2"}) {
			t.Errorf("attached %q, want linux-1/t7 then t2", got)
		}
	})
}
