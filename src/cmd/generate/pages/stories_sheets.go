package pages

import (
	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/catalog"
)

// value: the sheets' worlds: home as its stories draw it, reading what
// every Mac is signed in to (mac-mini to GitHub and Claude as the bot),
// and the fleet of This Mac alone.
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
	return []World{
		sheet("machines.sheet"),
		sheet("machines.sheet.first", thisMacAlone),
	}
}

func miniSignedIn(f *Scene) {
	f.Accounts.Machines = append(f.Accounts.Machines, accounts.Machine{Name: miniMac, Kind: accounts.KindMac, Accounts: []accounts.Account{
		own(catalog.ProviderGitHub, []string{"jterrazz-bot"}, []string{"repo"}, "", "github.com"),
		own(catalog.ProviderClaude, []string{"bot@jterrazz.com"}, []string{"max"}, "", "claude.ai"),
	}})
}
