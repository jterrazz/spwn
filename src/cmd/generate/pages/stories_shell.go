package pages

import (
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	model "github.com/jterrazz/studio-station/src/machines"
)

func shellWorlds() []World {
	return []World{
		{Name: "machines.restarting", Edit: infraRestarting, Pages: []string{"menubar"}},
	}
}

func infraRestarting(f *Scene) {
	infra := &f.All.Macs[1].VMs[0]
	infra.Restart = &model.RestartWalk{Step: 2, At: storyNow.Add(-8 * time.Second)}
	infra.BootAt = storyNow.Add(-20 * time.Second)
}

func (f Scene) shares() *accounts.AllMachines {
	all := accounts.AllMachines{Now: storyNow}
	for _, h := range f.All.Macs {
		mac := accounts.MacAccounts{Name: h.Name, ThisMac: h.ThisMac}
		for _, m := range f.Accounts.Machines {
			if m.Kind != accounts.KindVM || !hosts(h, m.Name) {
				continue
			}
			for _, a := range m.Accounts {
				if a.Shared != nil {
					mac.Shares = append(mac.Shares, accounts.Shared{Account: accounts.AccountOf(a.Shared.Name), VM: m.Name, At: a.Shared.At, Until: a.Shared.Until})
				}
			}
		}
		all.Macs = append(all.Macs, mac)
	}
	return &all
}

func hosts(h model.MacView, vm string) bool {
	for _, v := range h.VMs {
		if v.Name == vm {
			return true
		}
	}
	return false
}
