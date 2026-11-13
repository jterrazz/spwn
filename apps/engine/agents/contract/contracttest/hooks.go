package contracttest

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract"
)

type Recorded struct {
	// value: a payload as the app's hook hands it over, and the event it
	// reads as; the event's time is the adapter's clock, not compared.
	Name    string
	Payload []byte
	Want    agents.Event
}

type HooksSubject struct {
	Hooks contract.Hooks
	// value: whether the owner's own keys of the app's settings file are as they
	// were before station touched it.
	OwnersKept func(t *testing.T) bool
	Recorded   []Recorded
	Unknown    [][]byte
}

func Hooks(t *testing.T, open func(t *testing.T) HooksSubject) {
	ctx := context.Background()
	installed := func(t *testing.T, s HooksSubject) bool {
		t.Helper()
		ok, err := s.Hooks.Inspect(ctx)
		if err != nil {
			t.Fatalf("Inspect: %v", err)
		}
		return ok
	}

	t.Run("the app's agent is started by a program of its own", func(t *testing.T) {
		s := open(t)
		argv := s.Hooks.Launch()
		if len(argv) == 0 || argv[0] == "" {
			t.Fatalf("Launch() = %q, want the program its agent runs as", argv)
		}
	})

	t.Run("a hook installed is seen, once however often it is installed, and keeps the owner's keys", func(t *testing.T) {
		s := open(t)
		if installed(t, s) {
			t.Fatal("installed before any install")
		}
		for range 2 {
			if err := s.Hooks.Install(ctx); err != nil {
				t.Fatalf("Install: %v", err)
			}
		}
		if !installed(t, s) {
			t.Error("not installed after Install")
		}
		if !s.OwnersKept(t) {
			t.Error("Install touched the owner's own keys")
		}
	})

	t.Run("a hook removed is gone, the owner's keys kept, and removing again changes nothing", func(t *testing.T) {
		s := open(t)
		if err := s.Hooks.Install(ctx); err != nil {
			t.Fatalf("Install: %v", err)
		}
		for range 2 {
			if err := s.Hooks.Remove(ctx); err != nil {
				t.Fatalf("Remove: %v", err)
			}
		}
		if installed(t, s) {
			t.Error("still installed after Remove")
		}
		if !s.OwnersKept(t) {
			t.Error("Remove touched the owner's own keys")
		}
	})

	t.Run("each recorded payload reads as its event, and an unknown one is ignored", func(t *testing.T) {
		s := open(t)
		if len(s.Recorded) == 0 {
			t.Fatal("no recorded payload: the suite proves nothing")
		}
		for _, r := range s.Recorded {
			got, ok := s.Hooks.Event(ctx, r.Payload)
			if !ok {
				t.Errorf("%s: ignored", r.Name)
				continue
			}
			if got.At.IsZero() {
				t.Errorf("%s: read with no time", r.Name)
			}
			got.At = r.Want.At
			if got.App != s.Hooks.App() {
				t.Errorf("%s: app %q, want %q", r.Name, got.App, s.Hooks.App())
			}
			r.Want.App = s.Hooks.App()
			if !same(got, r.Want) {
				t.Errorf("%s: read %+v\nwant %+v", r.Name, got, r.Want)
			}
		}
		for _, p := range append(s.Unknown, []byte("not json"), nil) {
			if e, ok := s.Hooks.Event(ctx, p); ok {
				t.Errorf("%q read as %+v, want it ignored", p, e)
			}
		}
	})
}

func same(a, b agents.Event) bool {
	ca, cb := a.Change, b.Change
	a.Change, b.Change = nil, nil
	if a != b {
		return false
	}
	return (ca == nil && cb == nil) || (ca != nil && cb != nil && *ca == *cb)
}
