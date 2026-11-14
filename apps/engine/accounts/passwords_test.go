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
