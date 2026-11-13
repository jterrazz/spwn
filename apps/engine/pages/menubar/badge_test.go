package menubar

import (
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

func TestTheStatusItemCountsWhatNeedsYouAndSaysItsOneSignal(t *testing.T) {
	now := time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)
	server := func(state model.VMState, looping bool) model.VM {
		return model.VM{Name: "infra", Lifetime: model.AlwaysOn, State: state, Restarts: model.Restarts{Looping: looping}}
	}
	restarting := server(model.Running, false)
	restarting.Restart = &model.RestartWalk{Step: 2, At: now}
	fleet := func(vms ...model.VM) *model.AllMachines {
		return &model.AllMachines{Macs: []model.MacView{{Name: "jterrazz-macbook-pro", ThisMac: true}, {Name: "jterrazz-mac-mini", VMs: vms}}}
	}
	exposed := &ports.AllMachines{Macs: []ports.MacPorts{{Name: "jterrazz-mac-mini", Ports: []ports.Port{
		{Port: 5000, Reach: ports.Outside, Answer: ports.Answers},
		{Port: 8080, Reach: ports.Outside, Answer: ports.Unreachable},
	}}}}
	shared := func(left ...time.Duration) *accounts.AllMachines {
		mac := accounts.MacAccounts{Name: "jterrazz-macbook-pro", ThisMac: true}
		for _, l := range left {
			mac.Shares = append(mac.Shares, accounts.Shared{Account: "token", VM: "linux-1", At: now.Add(l - 12*time.Hour), Until: now.Add(l)})
		}
		return &accounts.AllMachines{Now: now, Macs: []accounts.MacAccounts{mac}}
	}
	waiting := model.VM{Name: "review-bot", Lifetime: model.Persistent, State: model.Running, Agent: &model.AgentView{State: "waiting", Ask: "PR #42 is ready"}}
	upkeep := &overview.Summary{Needs: []overview.Need{{Section: "apps", Subject: "updates", Text: "2 app updates"}, {Section: "settings", Subject: "firewall", Text: "Firewall: off"}}}
	cases := []struct {
		name string
		r    Reading
		want *page.Badge
	}{
		{"nothing read yet", Reading{Home: upkeep}, nil},
		{"only upkeep", Reading{Home: upkeep, All: fleet(server(model.Running, false))}, &page.Badge{Label: "all machines are fine"}},
		{"a share under 30 min", Reading{All: fleet(), Accounts: shared(22*time.Minute, 4*time.Hour)}, &page.Badge{Needs: 1, Label: "station, 1 thing needs you"}},
		{"an agent waiting on you", Reading{All: fleet(waiting), Accounts: shared(22 * time.Minute)}, &page.Badge{Needs: 2, Label: "station, 2 things need you"}},
		{"two shares ending", Reading{All: fleet(), Accounts: shared(22*time.Minute, 9*time.Minute)}, &page.Badge{Needs: 2, Label: "station, 2 things need you"}},
		{"a server down", Reading{All: fleet(server(model.Running, true)), Accounts: shared(22 * time.Minute)}, &page.Badge{Needs: 2, Down: true, Label: "station, infra down"}},
		{"a port exposed that answers", Reading{All: fleet(), Ports: exposed}, &page.Badge{Needs: 1, Down: true, Label: "station, port 5000 exposed"}},
		{"a server restarting", Reading{All: fleet(restarting), Accounts: shared(22 * time.Minute)}, &page.Badge{Needs: 1, Restarting: true, Label: "infra restarting, 2 of 4 checks"}},
		{"red wins over a restart", Reading{All: fleet(restarting), Ports: exposed}, &page.Badge{Needs: 1, Down: true, Label: "station, port 5000 exposed"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := badge(c.r)
			if (got == nil) != (c.want == nil) || got != nil && *got != *c.want {
				t.Fatalf("badge %+v, want %+v", got, c.want)
			}
		})
	}
}
