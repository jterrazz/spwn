package wire

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
	"github.com/jterrazz/studio-station/apps/engine/setup"
)

func TestNeverSleepsIsStillOnAfterAnotherPreferenceIsWritten(t *testing.T) {
	s := newStores(localtest.NewFS(), setup.Paths{Prefs: "/home/u/.config/station", State: "/home/u/.local/state/station"})

	if err := s.neverSleeps().Set(true); err != nil {
		t.Fatal(err)
	}
	if err := s.vmUser().Set("jterrazz"); err != nil {
		t.Fatal(err)
	}

	on, err := s.neverSleeps().Get()
	if err != nil || !on {
		t.Fatalf("never sleeps reads %v (%v), want on", on, err)
	}
}
