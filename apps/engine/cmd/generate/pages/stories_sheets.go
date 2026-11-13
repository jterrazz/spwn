package pages

import (
	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

// value: the sheets' worlds: home as its stories draw it, reading what
// every Mac is signed in to (mac-mini to GitHub and Claude as the bot),
// with linux-3's page over it; and the fleet of This Mac alone.
func sheetWorlds() []World {
	sheet := func(name string, edits ...func(*Scene)) World {
		return World{Name: name, Edit: func(f *Scene) {
			for _, e := range edits {
				e(f)
			}
			homeScene(f)
			miniSignedIn(f)
			f.SignedIn = true
		}, Pages: []string{"menubar"}}
	}
	mini := sheet("machines.sheet")
	mini.Pages = append(mini.Pages, "machine.linux-3")
	return []World{
		mini,
		sheet("machines.sheet.first", thisMacAlone),
	}
}

func miniSignedIn(f *Scene) {
	f.Accounts.Machines = append(f.Accounts.Machines, accounts.Machine{Name: miniMac, Kind: accounts.KindComputer, Accounts: []accounts.Account{
		own(catalog.ProviderGitHub, []string{"jterrazz-bot"}, []string{"repo"}, "", "github.com"),
		own(catalog.ProviderClaude, []string{"bot@jterrazz.com"}, []string{"max"}, "", "claude.ai"),
	}})
}
