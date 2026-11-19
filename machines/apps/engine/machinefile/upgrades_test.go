package machinefile_test

import (
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

func TestOnlyAMacBoundToTheFileUpgradesByItself(t *testing.T) {
	// invariant: a night runs on a Mac's engine for its bound file alone;
	// a VM, a Mac bound to none, or one bound to another file never moves
	// this file's apps by itself, and the last night is said only for it.
	night := machinefile.Night{At: time.Unix(1, 0), Machinefile: "m", Moved: []machinefile.Moved{{App: "app-a"}}}
	for _, c := range []struct {
		how     string
		mac     bool
		n       machinefile.Nights
		nightly bool
		last    bool
		file    string
	}{
		{how: "a VM", n: machinefile.Nights{Bound: "m", Night: night, Ran: true}, file: "m"},
		{how: "a Mac bound to none", mac: true},
		{how: "a Mac bound to another file", mac: true, n: machinefile.Nights{Bound: "other", Night: night, Ran: true}, file: "other"},
		{how: "a Mac bound to it, no night yet", mac: true, n: machinefile.Nights{Bound: "m"}, nightly: true, file: "m"},
		{how: "a Mac bound to it", mac: true, n: machinefile.Nights{Bound: "m", Night: night, Ran: true}, nightly: true, last: true, file: "m"},
	} {
		u := machinefile.UpgradesOf("mini", "m", c.mac, c.n)
		if u.Nightly != c.nightly || (u.Last != nil) != c.last || u.Sentence == "" || u.File != c.file {
			t.Errorf("%s: %+v", c.how, u)
		}
	}
}

func TestTheLastNightIsWordedByWhatItLeft(t *testing.T) {
	// invariant: a night that stopped or failed reads bad, one that left
	// a version waiting or an app for the owner reads warn, else ok.
	for _, c := range []struct {
		n    machinefile.Night
		tone verdict.Tone
		said string
	}{
		{machinefile.Night{Error: "the catalog has no package x"}, verdict.Bad, "It did not run to its end."},
		{machinefile.Night{Failed: []string{"app-a"}}, verdict.Bad, "1 update failed."},
		{machinefile.Night{Moved: []machinefile.Moved{{App: "app-a"}, {App: "app-b"}}, Waiting: []machinefile.Waiting{{App: "node"}}}, verdict.Warn, "2 apps moved."},
		{machinefile.Night{}, verdict.OK, "Nothing moved."},
	} {
		c.n.Machinefile = "m"
		u := machinefile.UpgradesOf("mini", "m", true, machinefile.Nights{Bound: "m", Night: c.n, Ran: true})
		if u.Last == nil || u.Last.Verdict.Tone != c.tone || u.Last.Verdict.Sentence != c.said {
			t.Errorf("%+v: %+v", c.n, u.Last)
		}
	}
}
