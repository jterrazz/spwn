package menubar

import (
	"testing"
	"time"

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
		return &model.AllMachines{Macs: []model.MacView{{Name: "macbook-pro", ThisMac: true}, {Name: "mac-mini", VMs: vms}}}
	}
	exposed := &ports.AllMachines{Macs: []ports.MacPorts{{Name: "mac-mini", Ports: []ports.Port{
		{Port: 5000, Reach: ports.Outside, Answer: ports.Answers},
		{Port: 8080, Reach: ports.Outside, Answer: ports.Unreachable},
	}}}}
	waiting := model.VM{Name: "review-bot", Lifetime: model.Persistent, State: model.Running, Agent: &model.AgentView{State: "waiting", Ask: "PR #42 is ready"}}
	upkeep := &overview.Summary{Needs: []overview.Need{{Section: "apps", Subject: "updates", Text: "2 app updates"}, {Section: "settings", Subject: "firewall", Text: "Firewall: off"}}}
	cases := []struct {
		name string
		r    Reading
		want *page.Badge
	}{
		{"nothing read yet", Reading{Home: upkeep}, nil},
		{"only upkeep", Reading{Home: upkeep, All: fleet(server(model.Running, false))}, &page.Badge{Label: "all machines are fine"}},
		{"an agent waiting on you", Reading{All: fleet(waiting)}, &page.Badge{Needs: 1, Label: "station, 1 thing needs you"}},
		{"a server down and an agent waiting", Reading{All: fleet(server(model.Running, true), waiting)}, &page.Badge{Needs: 2, Down: true, Label: "station, infra down"}},
		{"a port exposed that answers", Reading{All: fleet(), Ports: exposed}, &page.Badge{Needs: 1, Down: true, Label: "station, port 5000 exposed"}},
		{"a server restarting", Reading{All: fleet(restarting, waiting)}, &page.Badge{Needs: 1, Restarting: true, Label: "infra restarting, 2 of 4 checks"}},
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
