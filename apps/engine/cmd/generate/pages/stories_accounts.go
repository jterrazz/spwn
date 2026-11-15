package pages

import (
	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func accountsWorlds() []World {
	pages := []string{"accounts", "machine." + proName, "machine." + miniMac, "machine.linux-1"}
	return []World{
		{Name: "machines.accounts", Edit: macs().then(everyMacSignedIn), Pages: pages},
		{Name: "machines.accounts.signed-in", Edit: macs().then(everyMacSignedIn, npmSignedIn), Pages: []string{"accounts"}},
	}
}

// value: SPEC B5.8's shared fixture: what each Mac is signed in to, in
// its provider's words, npm signed out on This Mac.
func everyMacSignedIn(f *Scene) {
	for i, m := range f.Accounts.Machines {
		if m.Name == proName {
			f.Accounts.Machines[i].Accounts = append(f.Accounts.Machines[i].Accounts,
				own(catalog.ProviderTailscale, []string{"tail3e9a.ts.net"}, nil, "", ""),
				own(catalog.ProviderCloudflare, []string{"example.com"}, nil, "", ""),
				own(catalog.ProviderDocker, []string{"you"}, nil, "", ""))
		}
	}
	f.Accounts.Machines = append(f.Accounts.Machines, accounts.Machine{Name: miniMac, Kind: accounts.KindComputer, Accounts: []accounts.Account{
		own(catalog.ProviderGitHub, []string{"ci-bot"}, []string{"repo"}, "", "github.com"),
		own(catalog.ProviderClaude, []string{"you@example.com"}, []string{"max"}, "", "claude.ai"),
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
				f.Accounts.Machines[i].Accounts[j] = own(catalog.ProviderNPM, []string{"you"}, nil, "", "registry.npmjs.org")
			}
		}
	}
}
