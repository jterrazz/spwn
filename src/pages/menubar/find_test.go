package menubar

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/machine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/ports"
)

const (
	findPro  = "jterrazz-macbook-pro"
	findMini = "jterrazz-mac-mini"
	findAir  = "jterrazz-macbook-air"
	gib      = int64(1) << 30
)

var findNow = time.Date(2026, 11, 7, 15, 30, 0, 0, time.UTC)

func asksYesNo(gesture string, k machine.Kind) string {
	if gesture == "restart" && k == machine.Server {
		return LevelYesNo
	}
	return LevelNone
}

// value: the prototype's fleet: This Mac with linux-1 at work and agent-7f
// finished, mac-mini with infra healthy, linux-3 at work and review-bot
// waiting on its PR, macbook-air asleep; Vite of linux-1 shared, Postgres
// listening in linux-3; This Mac awake until 18:00 and its copy of Claude
// to linux-1 ending in 24 min.
func findFleet() FindReading {
	vm := func(name, mac string, life model.Lifetime, memory int64, agent *model.AgentView) model.VM {
		return model.VM{Name: name, Mac: mac, Lifetime: life, State: model.Running, Resources: model.Resources{MemoryBytes: memory * gib}, Agent: agent}
	}
	infra := vm("infra", findMini, model.AlwaysOn, 24, nil)
	infra.Want, infra.Guest, infra.SSH, infra.IP = "running", "connected", true, "192.168.64.7"
	infra.CheckStatus = model.CheckStatus{State: model.CheckPassing}
	all := model.AllMachines{Macs: []model.MacView{
		{Name: findPro, User: "jterrazz", ThisMac: true, Reachable: true, Laptop: true, Hardware: machine.Hardware{Memory: 48 * gib}, VMs: []model.VM{
			vm("linux-1", findPro, model.Throwaway, 8, &model.AgentView{State: "working"}),
			vm("agent-7f", findPro, model.Throwaway, 8, &model.AgentView{State: model.AgentFinished, Ended: findNow.Add(-12 * time.Minute), Reason: "tests pass"}),
		}},
		{Name: findMini, User: "jterrazz", Reachable: true, Hardware: machine.Hardware{Memory: 64 * gib}, VMs: []model.VM{
			infra,
			vm("linux-3", findMini, model.Throwaway, 8, &model.AgentView{State: "working"}),
			vm("review-bot", findMini, model.Permanent, 4, &model.AgentView{State: "waiting", Ask: "PR #42 is ready", Since: findNow.Add(-6 * time.Minute)}),
		}},
		{Name: findAir, User: "jterrazz", Laptop: true, Answered: findNow.Add(-2 * time.Hour)},
	}}
	vite := "https://" + findPro + ".tail.ts.net:40001"
	listed := ports.AllMachines{Macs: []ports.MacPorts{
		{Name: findPro, ThisMac: true, Reachable: true, Shared: []ports.SharedPort{
			{VM: "linux-1", GuestPort: 5173, HTTP: true, Process: "node", URLs: ports.URLSet{Public: vite}},
		}, Ports: []ports.Port{{Machine: machine.Ref{Name: "linux-1", On: findPro}, Port: 5173, Process: "node", Reach: ports.Tailnet}}},
		{Name: findMini, Reachable: true, Ports: []ports.Port{
			{Machine: machine.Ref{Name: "linux-3", On: findMini}, Port: 5432, Process: "postgres", Reach: ports.Tailnet},
		}},
	}}
	signedIn := accounts.Account{Provider: catalog.ProviderClaude, State: catalog.StateOK, Origin: accounts.OriginOwn}
	shares := accounts.AllMachines{Now: findNow, Macs: []accounts.MacAccounts{
		{Name: findPro, ThisMac: true, Accounts: []accounts.Account{signedIn},
			Shares: []accounts.Shared{{Account: catalog.ProviderClaude, VM: "linux-1", At: findNow.Add(-11 * time.Hour), Until: findNow.Add(24 * time.Minute)}}},
		{Name: findMini},
	}}
	held := &awake.Report{Engine: awake.EngineRunning, Mac: findPro, ThisMac: true,
		Session: &awake.Session{Lid: true, StartedAt: findNow.Add(-time.Hour), Until: time.Date(2026, 11, 7, 18, 0, 0, 0, time.UTC)}}
	return FindReading{Reading: Reading{All: &all, Ports: &listed, Accounts: &shares, Awake: held}, Now: findNow, Zone: time.UTC, Asks: asksYesNo}
}

// value: the same fleet once infra fell at 15:10.
func findFleetDown() FindReading {
	r := findFleet()
	infra := &r.All.Macs[1].VMs[0]
	infra.State, infra.Cause = model.Stopped, "out of memory"
	infra.Restarts = model.Restarts{Count: 5, Streak: 5, Looping: true}
	infra.CheckStatus = model.CheckStatus{State: model.CheckFailing, Since: time.Date(2026, 11, 7, 15, 10, 0, 0, time.UTC)}
	return r
}

type said struct{ title, where, verb, key, alt string }

func entries(t *testing.T, p page.Page, section string) []said {
	t.Helper()
	actions := map[string]page.Action{}
	for _, a := range p.Actions {
		actions[a.ID] = a
	}
	var out []said
	var walk func([]page.Item)
	walk = func(items []page.Item) {
		for _, it := range items {
			if it.Group != nil {
				walk(it.Group.Items)
				continue
			}
			r := it.Row
			s := said{title: r.Label, where: words(r.Detail), verb: actions[r.Verb].Label, key: actions[r.Verb].Key}
			if r.Quick != "" {
				s.alt = actions[r.Quick].Label
			}
			out = append(out, s)
		}
	}
	for _, s := range p.Sections {
		if s.ID == section {
			walk(s.Items)
		}
	}
	return out
}

func findRow(p page.Page, section, id string) *page.Row {
	for _, s := range p.Sections {
		if s.ID != section {
			continue
		}
		for _, it := range s.Items {
			if it.Row != nil && it.Row.ID == id {
				return it.Row
			}
		}
	}
	return nil
}

// invariant: SPEC B5.6: Find indexes each Mac in home's order on ⌘1, ⌘2,
// ⌘3, each VM with its state and its Mac, the shared ports and the
// listeners, the sections of each awake Mac and of every machine, and the
// actions available now, each with its verb, its key and its alternate.
func TestFindIndexesTheFleetsMachinesPortsSectionsAndActions(t *testing.T) {
	p := Find(findFleet(), page.Context{Machine: findPro, Kind: "mac"})
	if err := page.Check(p); err != nil {
		t.Fatalf("the Find index fails its checks: %v", err)
	}

	cases := []struct {
		section string
		want    []said
	}{
		{FindNeeds, []said{
			{"Review review-bot", "PR #42 is ready", "Review", "", ""},
			{"Extend Claude on linux-1", "Claude ends in 24 min", "Extend", "", ""},
		}},
		{FindMachines, []said{
			{"This Mac", "2 VMs | 28 GB free", "Open", "cmd+1", ""},
			{"mac-mini", "3 VMs | 24 GB free", "Open", "cmd+2", ""},
			{"macbook-air", "asleep | seen 2 hr ago", "Open", "cmd+3", ""},
			{"agent-7f", "finished | This Mac", "Open", "", "Open in Terminal"},
			{"linux-1", "working | This Mac", "Open", "", "Open in Terminal"},
			{"infra", "healthy | mac-mini", "Open", "", ""},
			{"linux-3", "working | mac-mini", "Open", "", "Open in Terminal"},
			{"review-bot", "waiting | mac-mini", "Open", "", "Open in Terminal"},
		}},
		{FindPorts, []said{
			{"Vite", "linux-1:5173", "Open", "", "Copy link"},
			{"Postgres", "linux-3:5432", "Share", "", ""},
		}},
		{FindSections, []said{
			{"Ports of This Mac", "", "Open", "", ""},
			{"Activity of This Mac", "", "Open", "", ""},
			{"Ports of mac-mini", "", "Open", "", ""},
			{"Activity of mac-mini", "", "Open", "", ""},
			{"Ports", "All machines", "Open", "cmd+p", ""},
			{"Accounts", "All machines", "Open", "cmd+u", ""},
			{"Awake", "All Macs", "Open", "cmd+e", ""},
		}},
		{FindActions, []said{
			{"Keep This Mac awake 4 hr", "", "Keep awake", "", ""},
			{"New VM on This Mac", "28 GB free", "New VM", "cmd+n", ""},
			{"New VM on mac-mini", "24 GB free", "New VM", "", ""},
			{"Restart infra…", "asks first", "Restart", "", ""},
			{"Clean up throwaways on This Mac…", "", "Clean up", "", ""},
		}},
	}
	for _, c := range cases {
		if got := entries(t, p, c.section); !slices.Equal(got, c.want) {
			t.Errorf("%s:\n got %+v\nwant %+v", c.section, got, c.want)
		}
	}
	if got, want := entries(t, p, FindVerbs), []said{
		{"Open agent-7f in Terminal", "", "Open in Terminal", "cmd+enter", ""},
		{"Share a port of agent-7f", "", "Share", "", ""},
		{"Lend an account to agent-7f", "", "Lend", "", ""},
		{"Open linux-1 in Terminal", "", "Open in Terminal", "cmd+enter", ""},
		{"Share a port of linux-1", "", "Share", "", ""},
		{"Lend an account to linux-1", "", "Lend", "", ""},
		{"Share a port of infra", "", "Share", "", ""},
		{"Restart infra", "", "Restart", "", ""},
		{"Open linux-3 in Terminal", "", "Open in Terminal", "cmd+enter", ""},
		{"Share a port of linux-3", "", "Share", "", ""},
		{"Open review-bot in Terminal", "", "Open in Terminal", "cmd+enter", ""},
		{"Share a port of review-bot", "", "Share", "", ""},
	}; !slices.Equal(got, want) {
		t.Errorf("the VMs' verbs:\n got %+v\nwant %+v", got, want)
	}
	if r := findRow(p, FindMachines, "linux-1"); r == nil || r.Nav != "menubar.machine:linux-1" || !strings.Contains(words(r.Cells[0]), "agent") {
		t.Errorf("linux-1's entry %+v, want its page and the tag agent", r)
	}
	if r := findRow(p, FindPorts, "listening:linux-3:5432"); r == nil || !strings.Contains(words(r.Cells[0]), "| 5432 |") {
		t.Errorf("Postgres's entry %+v, want it found by its number", r)
	}
}

// invariant: SPEC A6 [find.down]: while infra is down Find leads with
// Restart infra, run at once, and says since when it is down.
func TestFindLeadsWithRestartInfraWhileItIsDown(t *testing.T) {
	p := Find(findFleetDown(), page.Context{Machine: findPro, Kind: "mac"})
	if err := page.Check(p); err != nil {
		t.Fatalf("the Find index fails its checks: %v", err)
	}
	needs := entries(t, p, FindNeeds)
	if len(needs) != 3 || needs[0] != (said{"Restart infra", "down since 15:10", "Restart", "", ""}) || needs[1].title != "Review review-bot" {
		t.Errorf("Find's needs %+v, want Restart infra first, then Review review-bot", needs)
	}
	restart := findRow(p, FindNeeds, "down:infra")
	for _, a := range p.Actions {
		if restart != nil && a.ID == restart.Verb && (a.Confirm != nil || a.Run == nil || a.Run.Call != CallRestartVM) {
			t.Errorf("Restart infra from Find runs %+v, want machines.restart at once", a)
		}
	}
	if r := findRow(p, FindMachines, "infra"); r == nil || words(r.Detail) != "down since 15:10 | mac-mini" || r.Glyph.Signal != page.SignalFailing {
		t.Errorf("infra's entry %+v, want down since 15:10 on mac-mini, in red", r)
	}
	if r := findRow(p, FindMachines, findMini); r == nil || r.Glyph.Signal != page.SignalFailing {
		t.Errorf("mac-mini's entry %+v, want it red while its server is down", r)
	}
	for _, e := range entries(t, p, FindActions) {
		if strings.HasPrefix(e.title, "Restart infra") && e != (said{"Restart infra", "down", "Restart", "", ""}) {
			t.Errorf("the restart action reads %+v, want Restart infra, down, asking nothing", e)
		}
	}
}

// invariant: nothing read yet is Find's frame, loading.
func TestFindIsLoadingUntilTheMachinesAreRead(t *testing.T) {
	if p := Find(FindReading{}, page.Context{Machine: findPro}); p.State != page.Loading || p.ID != FindID {
		t.Errorf("Find before any read is %+v, want its loading frame", p)
	}
}

// invariant: a keep-awake from Find never shortens the term held: past
// 4 hr left it extends the end by 4 hr.
func TestFindExtendsAwakeWhenMoreThanFourHoursAreLeft(t *testing.T) {
	r := findFleet()
	r.Awake.Session.Until = time.Date(2026, 11, 7, 21, 0, 0, 0, time.UTC)
	p := Find(r, page.Context{Machine: findPro, Kind: "mac"})
	if got := entries(t, p, FindActions)[0]; got.title != "Extend awake to 01:00" {
		t.Errorf("Find's awake action reads %q, want Extend awake to 01:00", got.title)
	}
	for _, a := range p.Actions {
		if a.ID == "awake-for" && (a.Run.In["Until"] != `"2026-11-08T01:00:00Z"` || a.Run.In["Lid"] != "true") {
			t.Errorf("the extension asks %v, want until 01:00 with the lid closed as held", a.Run.In)
		}
	}
}
