package pages

import (
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/catalog"
)

func accountsWorlds() []World {
	pages := []string{"accounts", "machine." + proName, "machine." + miniMac, "machine.linux-1"}
	return []World{
		{Name: "machines.accounts", Edit: macs().then(everyMacSignedIn), Pages: pages},
		{Name: "machines.accounts.signed-in", Edit: macs().then(everyMacSignedIn, npmSignedIn), Pages: []string{"accounts"}},
		{Name: "machines.accounts.extended", Edit: macs().then(everyMacSignedIn, claudeExtended), Pages: []string{"machine.linux-1"}},
		{Name: "machines.accounts.no-loan", Edit: macs().then(everyMacSignedIn, nothingLentTo("review-bot")), Pages: []string{"machine.review-bot"}},
	}
}

// value: Extend pressed on Claude's copy to linux-1: 4 hr more than the
// 24 min it had left.
func claudeExtended(f *Scene) {
	for i, m := range f.Accounts.Machines {
		for j, a := range m.Accounts {
			if m.Name == "linux-1" && a.Provider == catalog.ProviderClaude && a.Shared != nil {
				loan := *a.Shared
				loan.Until = loan.Until.Add(4 * time.Hour)
				f.Accounts.Machines[i].Accounts[j].Shared = &loan
			}
		}
	}
}

func nothingLentTo(vm string) func(*Scene) {
	return func(f *Scene) {
		for i, m := range f.Accounts.Machines {
			if m.Name == vm {
				f.Accounts.Machines[i].Accounts = nil
			}
		}
	}
}

// value: SPEC B5.8's shared fixture: what each Mac is signed in to, in
// its provider's words, npm signed out on This Mac.
func everyMacSignedIn(f *Scene) {
	for i, m := range f.Accounts.Machines {
		if m.Name == proName {
			f.Accounts.Machines[i].Accounts = append(f.Accounts.Machines[i].Accounts,
				own(catalog.ProviderTailscale, []string{"tail3e9a.ts.net"}, nil, "", ""),
				own(catalog.ProviderCloudflare, []string{"jterrazz.com"}, nil, "", ""),
				own(catalog.ProviderDocker, []string{"jterrazz"}, nil, "", ""))
		}
	}
	f.Accounts.Machines = append(f.Accounts.Machines, accounts.Machine{Name: miniMac, Kind: accounts.KindMac, Accounts: []accounts.Account{
		own(catalog.ProviderGitHub, []string{"jterrazz-bot"}, []string{"repo"}, "", "github.com"),
		own(catalog.ProviderClaude, []string{"admin@jterrazz.com"}, []string{"max"}, "", "claude.ai"),
		own(catalog.ProviderTailscale, []string{"tail3e9a.ts.net"}, nil, "", ""),
	}})
}

func npmSignedIn(f *Scene) {
	for i, m := range f.Accounts.Machines {
		if m.Name != proName {
			continue
		}
		for j, a := range m.Accounts {
			if a.Provider == catalog.ProviderNPM {
				f.Accounts.Machines[i].Accounts[j] = own(catalog.ProviderNPM, []string{"jterrazz"}, nil, "", "registry.npmjs.org")
			}
		}
	}
}
