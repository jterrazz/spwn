package contracttest

import (
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

const (
	Mac     = "mac-mini"
	LAN     = "192.168.1.20"
	Refusal = catalog.Sudo + ": a password is required"
)

type UnlockerSubject struct {
	Unlocker contract.Unlocker
	// value: the state of Mac for the calls from now on; at pre-boot it is
	// off the tailnet, on your LAN at lan.
	AtPreBoot          func(mac, lan string)
	Offline, Answering func(mac string)
	Refusing           func(mac, detail string)
	// value: the password sessions macOS's pre-boot prompt was opened for.
	Sessions func(mac string) int
	// value: the planned restarts the Mac received.
	Restarts func(mac string) int
}

func Unlocker(t *testing.T, subject func(t *testing.T) UnlockerSubject) {
	t.Run("a Mac waiting at pre-boot gets the password session of macOS's own prompt", func(t *testing.T) {
		s := subject(t)
		s.AtPreBoot(Mac, LAN)
		if err := s.Unlocker.Unlock(t.Context(), Mac, LAN); err != nil {
			t.Fatal(err)
		}
		if got := s.Sessions(Mac); got != 1 {
			t.Errorf("%d password sessions opened, want 1", got)
		}
	})
	t.Run("a Mac at pre-boot is off the tailnet: dialed by its name alone it does not answer", func(t *testing.T) {
		s := subject(t)
		s.AtPreBoot(Mac, LAN)
		if err := s.Unlocker.Unlock(t.Context(), Mac, ""); !errors.Is(err, contract.ErrUnreachable) {
			t.Fatalf("unlocking a Mac at pre-boot by its name returned %v, want ErrUnreachable", err)
		}
		if got := s.Sessions(Mac); got != 0 {
			t.Errorf("%d password sessions opened, want none", got)
		}
	})
	t.Run("a Mac that does not answer is unreachable, and no session opens", func(t *testing.T) {
		s := subject(t)
		s.Offline(Mac)
		if err := s.Unlocker.Unlock(t.Context(), Mac, LAN); !errors.Is(err, contract.ErrUnreachable) {
			t.Fatalf("unlocking an offline Mac returned %v, want ErrUnreachable", err)
		}
		if got := s.Sessions(Mac); got != 0 {
			t.Errorf("%d password sessions opened for an offline Mac, want none", got)
		}
	})
	t.Run("a Mac that answers receives its planned restart", func(t *testing.T) {
		s := subject(t)
		s.Answering(Mac)
		if err := s.Unlocker.PlannedRestart(t.Context(), Mac); err != nil {
			t.Fatal(err)
		}
		if got := s.Restarts(Mac); got != 1 {
			t.Errorf("the Mac received %d restarts, want 1", got)
		}
	})
	t.Run("a Mac that refuses the restart says why", func(t *testing.T) {
		s := subject(t)
		s.Refusing(Mac, Refusal)
		var refused *contract.RestartRefusedError
		if err := s.Unlocker.PlannedRestart(t.Context(), Mac); !errors.As(err, &refused) || refused.Detail != Refusal {
			t.Fatalf("a refused restart returned %v, want a refusal saying %q", err, Refusal)
		}
		if got := s.Restarts(Mac); got != 0 {
			t.Errorf("a refusing Mac received %d restarts, want none", got)
		}
	})
}
