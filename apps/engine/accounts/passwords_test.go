package accounts_test

import (
	"strings"
	"testing"

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

func TestAProgramOffersAddFromASignInExactlyWhenItsDeliveryPrintsItsToken(t *testing.T) {
	offered := accounts.VaultPrograms()
	if len(offered) != len(catalog.Deliveries) {
		t.Fatalf("%d programs offered; the catalog delivers to %d", len(offered), len(catalog.Deliveries))
	}
	for _, p := range offered {
		d, _ := p.Program.Delivery()
		if p.Captures != (len(d.Capture) > 0) {
			t.Errorf("%s captures = %v; its delivery prints a token: %v", p.Program, p.Captures, len(d.Capture) > 0)
		}
		if p.Label == "" {
			t.Errorf("%s has no label", p.Program)
		}
	}
}

func TestAnEntryCountsWhatItsRemovalEnds(t *testing.T) {
	machine := accounts.VaultGrant{ToKind: accounts.ToMachine, To: "linux-1"}
	other := accounts.VaultGrant{ToKind: accounts.ToMachine, To: "linux-2"}
	agent := accounts.VaultGrant{ToKind: accounts.ToAgent, To: "docs-bot", Workspace: "linux-1"}
	for name, c := range map[string]struct {
		grants []accounts.VaultGrant
		reach  accounts.EntryReach
		want   string
	}{
		"none":              {nil, accounts.ReachNotGranted, "Nothing else ends"},
		"one machine":       {[]accounts.VaultGrant{machine}, accounts.ReachGranted, "Its grants end: 1 machine"},
		"machines":          {[]accounts.VaultGrant{machine, other}, accounts.ReachGranted, "Its grants end: 2 machines"},
		"one agent":         {[]accounts.VaultGrant{agent}, accounts.ReachGranted, "Its grants end: 1 agent"},
		"a machine, agents": {[]accounts.VaultGrant{machine, agent, agent}, accounts.ReachGranted, "Its grants end: 1 machine and 2 agents"},
	} {
		got := accounts.DescribeEntry(accounts.VaultEntry{Grants: c.grants})
		if got.RemoveSentence != c.want || got.Reach != c.reach {
			t.Errorf("%s: %q, %q; want %q, %q", name, got.RemoveSentence, got.Reach, c.want, c.reach)
		}
	}
}
