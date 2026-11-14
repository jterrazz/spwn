package accounts_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

var now = time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)

func twoMacs() accounts.AllMachines {
	return accounts.AllMachines{Now: now, Macs: []accounts.MacAccounts{
		{Name: "jterrazz-macbook-pro", ThisMac: true, Accounts: []accounts.Account{
			{Provider: catalog.ProviderGitHub, State: catalog.StateOK, Identity: []string{"jterrazz"}, Scopes: []string{"repo", "workflow"}},
			{Provider: catalog.ProviderCloudflare, State: catalog.StateBroken, Reason: accounts.ReasonRefused, Identity: []string{accounts.Unknown}},
			{Provider: catalog.ProviderClaude, State: catalog.StateOK, Identity: []string{"admin@jterrazz.com"}, Scopes: []string{"max"}},
			{Provider: catalog.ProviderDocker, State: catalog.StateOK},
			{Provider: catalog.ProviderGit, State: catalog.StateOK, Identity: []string{"Jean-Baptiste Terrazzoni"}},
			{Provider: catalog.ProviderSSH, State: catalog.StateOK, Identity: []string{"1 key"}},
		}, Shares: []accounts.Shared{
			{Account: catalog.ProviderGitHub, VM: "linux-1", At: now.Add(-8 * time.Hour), Until: now.Add(4 * time.Hour)},
			{Account: catalog.ProviderClaude, VM: "linux-1", At: now.Add(-3 * time.Hour), Until: now.Add(22 * time.Minute)},
		}},
		{Name: "jterrazz-mac-mini", Accounts: []accounts.Account{
			{Provider: catalog.ProviderGitHub, State: catalog.StateOK, Identity: []string{"jterrazz-agent"}, Scopes: []string{"repo"}},
			{Provider: catalog.ProviderCloudflare, State: catalog.StateAbsent, Reason: accounts.ReasonUnset},
		}, Shares: []accounts.Shared{
			{Account: catalog.ProviderClaude, VM: "linux-3", At: now.Add(-3 * time.Hour), Until: now.Add(9 * time.Hour)},
		}},
		{Name: "jterrazz-macbook-air", Silent: "jterrazz-macbook-air does not answer"},
	}}
}

func TestTheSignedOutAccountsOfEveryMacComeFirstThisMacsFirst(t *testing.T) {
	got := twoMacs().Read().SignedOut
	want := []accounts.SignIn{
		{Mac: "jterrazz-macbook-pro", ThisMac: true, Provider: catalog.ProviderCloudflare, State: catalog.StateBroken, Reason: accounts.ReasonRefused},
		{Mac: "jterrazz-mac-mini", Provider: catalog.ProviderCloudflare, State: catalog.StateAbsent, Reason: accounts.ReasonUnset},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("signed out: %+v, want %+v", got, want)
	}
}

func TestTheSignedInAccountsAreGroupedByMacInTheProvidersOwnWords(t *testing.T) {
	got := twoMacs().Read().SignedIn
	words := map[string][][]string{}
	for _, on := range got {
		for _, s := range on.Accounts {
			words[on.Mac] = append(words[on.Mac], append([]string{s.Provider}, s.Words...))
		}
	}
	want := map[string][][]string{
		"jterrazz-macbook-pro": {{catalog.ProviderGitHub, "jterrazz", "repo, workflow"}, {catalog.ProviderClaude, "Max plan"}, {catalog.ProviderDocker}},
		"jterrazz-mac-mini":    {{catalog.ProviderGitHub, "jterrazz-agent", "repo"}},
	}
	if len(got) != 2 || got[0].Mac != "jterrazz-macbook-pro" || !got[0].ThisMac || !reflect.DeepEqual(words, want) {
		t.Fatalf("signed in: %+v, want This Mac then the mini with %v", got, want)
	}
}

func TestAPartTheReadDidNotGiveIsLeftOutOfTheWords(t *testing.T) {
	for _, c := range []struct {
		account accounts.Account
		want    []string
	}{
		{accounts.Account{Provider: catalog.ProviderGitHub, Identity: []string{"jterrazz"}}, []string{"jterrazz"}},
		{accounts.Account{Provider: catalog.ProviderClaude, Identity: []string{"admin@jterrazz.com"}}, nil},
		{accounts.Account{Provider: catalog.ProviderClaude, Scopes: []string{"ultra"}}, []string{"ultra"}},
		{accounts.Account{Provider: catalog.ProviderCloudflare, Identity: []string{accounts.Unknown}}, nil},
	} {
		if got := accounts.Words(c.account); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s %v %v: words %q, want %q", c.account.Provider, c.account.Identity, c.account.Scopes, got, c.want)
		}
	}
}

func TestEverySharedCopyCarriesItsTimeLeftAndTotalTheShortOneFirst(t *testing.T) {
	got := twoMacs().Read().Shared
	want := []accounts.SharedCopy{
		{Mac: "jterrazz-macbook-pro", VM: "linux-1", Provider: catalog.ProviderClaude, Scope: "Claude Code inference", Left: 22 * time.Minute, Total: 3*time.Hour + 22*time.Minute, Short: true},
		{Mac: "jterrazz-macbook-pro", VM: "linux-1", Provider: catalog.ProviderGitHub, Scope: "GitHub browser session", Left: 4 * time.Hour, Total: 12 * time.Hour},
		{Mac: "jterrazz-mac-mini", VM: "linux-3", Provider: catalog.ProviderClaude, Scope: "Claude Code inference", Left: 9 * time.Hour, Total: 12 * time.Hour},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("shared: %+v\nwant %+v", got, want)
	}
}

func TestAMacThatDoesNotAnswerIsNamedAndTheOthersAreRead(t *testing.T) {
	r := twoMacs().Read()
	if want := []accounts.Silent{{Mac: "jterrazz-macbook-air", Why: "jterrazz-macbook-air does not answer"}}; !reflect.DeepEqual(r.Silent, want) {
		t.Fatalf("silent: %+v, want %+v", r.Silent, want)
	}
}

func TestAVMsAccountsAreTheCopiesItsMacSharedWithIt(t *testing.T) {
	r := twoMacs().Of("jterrazz-macbook-pro", "linux-1").Read()
	if len(r.SignedOut) != 0 || len(r.SignedIn) != 0 || len(r.Shared) != 2 || r.Shared[0].VM != "linux-1" || r.Shared[1].VM != "linux-1" {
		t.Fatalf("linux-1's accounts: %+v", r)
	}
	mini := twoMacs().Of("jterrazz-mac-mini", "").Read()
	if len(mini.SignedOut) != 1 || len(mini.SignedIn) != 1 || mini.SignedIn[0].Mac != "jterrazz-mac-mini" || len(mini.Shared) != 1 {
		t.Fatalf("the mini's accounts: %+v", mini)
	}
}

func TestAMacNotReadSaysItsErrorWithItsTokenWithheld(t *testing.T) {
	token := "s" + "k-" + strings.Repeat("Ab3dE", 8)
	r := accounts.AllMachines{Macs: []accounts.MacAccounts{{Name: "jterrazz-mac-mini", Silent: "refused: token " + token}}}.Read()
	if len(r.Silent) != 1 || r.Silent[0].Why != "refused: token withheld" {
		t.Errorf("silent: %+v", r.Silent)
	}
}
