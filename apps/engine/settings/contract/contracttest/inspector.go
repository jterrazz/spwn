package contracttest

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

type InspectorSubject struct {
	Inspector contract.Inspector
	// value: a protection this machine has on, and one it has off; none on
	// an Inspector that reads no protection.
	On, Off catalog.Fact
	// value: a fact this machine cannot look at.
	Unreadable catalog.Fact
	// value: a fact this machine has, read by a command.
	Present catalog.Fact
}

func Inspector(t *testing.T, subject func(t *testing.T) InspectorSubject) {
	ctx := context.Background()
	read := func(t *testing.T, i contract.Inspector, f catalog.Fact) (catalog.State, string) {
		t.Helper()
		got, err := i.Read(ctx, f)
		if err != nil {
			t.Fatalf("Read(%s): %v", f, err)
		}
		return got.State, got.Value
	}

	t.Run("a protection on reads on, one off reads off", func(t *testing.T) {
		s := subject(t)
		if s.On == "" {
			t.Skip("this Inspector reads no protection")
		}
		if state, value := read(t, s.Inspector, s.On); state != catalog.StateOK || value != "on" {
			t.Errorf("Read(%s) = %s %q; want ok on", s.On, state, value)
		}
		if state, value := read(t, s.Inspector, s.Off); state != catalog.StateAbsent || value != "off" {
			t.Errorf("Read(%s) = %s %q; want absent off", s.Off, state, value)
		}
	})

	t.Run("a read past its context is an error, never a value", func(t *testing.T) {
		s := subject(t)
		done, cancel := context.WithCancel(ctx)
		cancel()
		if s.Present == "" {
			t.Skip("this Inspector names no fact it reads")
		}
		if got, err := s.Inspector.Read(done, s.Present); err == nil && got.State == catalog.StateOK {
			t.Errorf("Read(%s) on an ended context = %+v; want an error", s.Present, got)
		}
	})

	t.Run("a fact the machine cannot look at is unknown, never off", func(t *testing.T) {
		s := subject(t)
		if state, value := read(t, s.Inspector, s.Unreadable); state != catalog.StateUnknown || value == "off" {
			t.Errorf("Read(%s) = %s %q; want unknown", s.Unreadable, state, value)
		}
	})
}
