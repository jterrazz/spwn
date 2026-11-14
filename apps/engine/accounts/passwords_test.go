package accounts_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
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
