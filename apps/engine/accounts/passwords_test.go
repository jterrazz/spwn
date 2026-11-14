package accounts_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func TestThePasswordsVerdictCountsWhatIsGranted(t *testing.T) {
	granted := accounts.VaultEntry{ID: "a", Grants: []accounts.VaultGrant{{To: "linux-1"}, {To: "linux-2"}}}
	kept := accounts.VaultEntry{ID: "b"}
	for name, c := range map[string]struct {
		entries []accounts.VaultEntry
		want    string
	}{
		"none":            {nil, "No password kept yet"},
		"one entry twice": {[]accounts.VaultEntry{granted, kept}, "2 kept, 1 granted"},
		"none granted":    {[]accounts.VaultEntry{kept}, "1 kept, 0 granted"},
	} {
		if got := accounts.SummarizeVault(c.entries).Sentence; got != c.want {
			t.Errorf("%s: %q; want %q", name, got, c.want)
		}
	}
}

func TestAGrantLastsAMinuteToSevenDaysAndOnlyAMachineKeepsItUntilRemoved(t *testing.T) {
	for _, c := range []struct {
		to   accounts.GrantTo
		term time.Duration
		want bool
	}{
		{accounts.ToAgent, 0, false},
		{accounts.ToMachine, 0, true},
		{accounts.ToAgent, time.Minute, true},
		{accounts.ToAgent, time.Minute - time.Second, false},
		{accounts.ToAgent, 7 * 24 * time.Hour, true},
		{accounts.ToMachine, 7*24*time.Hour + time.Second, false},
		{accounts.ToMachine, -time.Hour, false},
	} {
		if got := accounts.TermInBounds(c.to, c.term); got != c.want {
			t.Errorf("TermInBounds(%s, %s) = %v; want %v", c.to, c.term, got, c.want)
		}
	}
}

func TestEveryProgramStationSignsInHasAWayInAndOutOfAVM(t *testing.T) {
	for _, p := range accounts.Programs() {
		d, ok := p.Delivery()
		if !ok {
			t.Fatalf("%s has no delivery", p)
		}
		if (d.Env == "") == (d.Inject == "" || d.Remove == "") {
			t.Errorf("%s: an environment variable or both lines, never neither nor both: %+v", p, d)
		}
	}
}

func TestARemovalSaysAnEnvironmentFileProgramLeavesNewShellsOnly(t *testing.T) {
	for _, d := range catalog.Deliveries {
		got := accounts.RevokedSentence(accounts.Program(d.Program))
		if strings.Contains(got, "New shells no longer see it") != (d.Env != "") {
			t.Errorf("%s (variable %q): %q", d.Program, d.Env, got)
		}
	}
}
