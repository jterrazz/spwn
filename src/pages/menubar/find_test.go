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

func asksYesNo(gesture string, k machine.Kind, _ ...string) string {
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

type findSaid struct{ title, where, verb, key, alt string }

func entries(t *testing.T, p page.Page, section string) []findSaid {
	t.Helper()
	actions := map[string]page.Action{}
	for _, a := range p.Actions {
		actions[a.ID] = a
	}
	var out []findSaid
	var walk func([]page.Item)
	walk = func(items []page.Item) {
		for _, it := range items {
			if it.Group != nil {
				walk(it.Group.Items)
				continue
			}
			r := it.Row
			s := findSaid{title: r.Label, where: words(r.Detail), verb: actions[r.Verb].Label, key: actions[r.Verb].Key}
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
		want    []findSaid
	}{
		{FindNeeds, []findSaid{
			{"Review review-bot", "PR #42 is ready", "Review", "", ""},
			{"Extend Claude on linux-1", "Claude ends in 24 min", "Extend", "", ""},
		}},
		{FindMachines, []findSaid{
			{"This Mac", "2 VMs | 28 GB free", "Open", "cmd+1", ""},
			{"mac-mini", "3 VMs | 24 GB free", "Open", "cmd+2", ""},
			{"macbook-air", "asleep | seen 2 hr ago", "Open", "cmd+3", ""},
			{"agent-7f", "finished | This Mac", "Open", "", "Open in Terminal"},
			{"linux-1", "working | This Mac", "Open", "", "Open in Terminal"},
			{"infra", "healthy | mac-mini", "Open", "", ""},
			{"linux-3", "working | mac-mini", "Open", "", "Open in Terminal"},
			{"review-bot", "waiting | mac-mini", "Open", "", "Open in Terminal"},
		}},
		{FindPorts, []findSaid{
			{"Vite", "linux-1:5173", "Open", "", "Copy link"},
			{"Postgres", "linux-3:5432", "Share", "", ""},
		}},
		{FindSections, []findSaid{
			{"Ports of This Mac", "", "Open", "", ""},
			{"Activity of This Mac", "", "Open", "", ""},
			{"Ports of mac-mini", "", "Open", "", ""},
			{"Activity of mac-mini", "", "Open", "", ""},
			{"Ports", "All machines", "Open", "cmd+p", ""},
			{"Accounts", "All machines", "Open", "cmd+u", ""},
			{"Awake", "All Macs", "Open", "cmd+e", ""},
		}},
		{FindActions, []findSaid{
			{"Keep This Mac awake 4 hr", "", "Keep awake", "", ""},
			{"New VM on This Mac", "28 GB free", "New VM", "cmd+n", ""},
			{"New VM on mac-mini", "24 GB free", "New VM", "", ""},
			{"Restart infra", "asks first", "Restart", "", ""},
			{"Clean up throwaways on This Mac", "", "Clean up", "", ""},
		}},
	}
	for _, c := range cases {
		if got := entries(t, p, c.section); !slices.Equal(got, c.want) {
			t.Errorf("%s:\n got %+v\nwant %+v", c.section, got, c.want)
		}
	}
	if got, want := entries(t, p, FindVerbs), []findSaid{
		{"Open agent-7f in Terminal", "", "Open in Terminal", "cmd+enter", ""},
		{"Share a port of agent-7f", "", "Share", "", ""},
		{"Lend an account to agent-7f", "", "Run", "", ""},
		{"Open linux-1 in Terminal", "", "Open in Terminal", "cmd+enter", ""},
		{"Share a port of linux-1", "", "Share", "", ""},
		{"Lend an account to linux-1", "", "Run", "", ""},
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
	if len(needs) != 3 || needs[0] != (findSaid{"Restart infra", "down since 15:10", "Restart", "", ""}) || needs[1].title != "Review review-bot" {
		t.Errorf("Find's needs %+v, want Restart infra first, then Review review-bot", needs)
	}
	restart := findRow(p, FindNeeds, "down:infra")
	if restart == nil || restart.Mark != page.Broken {
		t.Errorf("infra's need %+v, want the red mark of a broken machine", restart)
	}
	if r := findRow(p, FindNeeds, "waiting:review-bot"); r == nil || r.Mark != page.Attention {
		t.Errorf("review-bot's need %+v, want the orange mark of a wait", r)
	}
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
		if strings.HasPrefix(e.title, "Restart infra") && e != (findSaid{"Restart infra", "down", "Restart", "", ""}) {
			t.Errorf("the restart action reads %+v, want Restart infra, down", e)
		}
	}
	action := findRow(p, FindActions, "restart:infra")
	for _, a := range p.Actions {
		if action != nil && a.ID == action.Verb && (a.Confirm != nil || a.Run == nil || a.Run.Call != CallRestartVM) {
			t.Errorf("Restart infra among the actions runs %+v, want machines.restart asking nothing", a)
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

// invariant: ⌘N goes to the roomiest Mac, even when it is not This Mac.
func TestFindGivesNewVMToTheRoomiestMac(t *testing.T) {
	r := findFleet()
	r.All.Macs[0].Hardware.Memory = 24 * gib
	p := Find(r, page.Context{Machine: findPro, Kind: "mac"})
	var got []findSaid
	for _, e := range entries(t, p, FindActions) {
		if strings.HasPrefix(e.title, "New VM") {
			got = append(got, e)
		}
	}
	if len(got) != 2 || got[0].title != "New VM on mac-mini" || got[0].key != "cmd+n" || got[1].title != "New VM on This Mac" || got[1].key != "" {
		t.Errorf("Find's New VM actions %+v, want mac-mini first on ⌘N, then This Mac with no key", got)
	}
}

// invariant: an asleep Mac offers no clean-up, finished throwaways or not.
func TestFindOffersNoCleanUpOnAnAsleepMac(t *testing.T) {
	r := findFleet()
	r.All.Macs[2].VMs = []model.VM{{Name: "agent-9a", Mac: findAir, Lifetime: model.Throwaway, State: model.Running,
		Agent: &model.AgentView{State: model.AgentFinished, Ended: findNow.Add(-3 * time.Hour)}}}
	p := Find(r, page.Context{Machine: findPro, Kind: "mac"})
	if row := findRow(p, FindActions, "cleanup:"+findAir); row != nil {
		t.Errorf("the asleep macbook-air offers %+v, want no clean-up", row)
	}
	if row := findRow(p, FindActions, "cleanup:"+findPro); row == nil {
		t.Errorf("This Mac lost its clean-up, want Clean up throwaways on This Mac")
	}
}

// invariant: an action that asks first carries its question, which the app
// draws as the label's ellipsis: "Restart infra…" and "Clean up throwaways
// on This Mac…" in the prototype.
func TestFindsActionsThatAskFirstCarryTheirQuestion(t *testing.T) {
	p := Find(findFleet(), page.Context{Machine: findPro, Kind: "mac"})
	actions := map[string]page.Action{}
	for _, a := range p.Actions {
		actions[a.ID] = a
	}
	for _, id := range []string{"restart:infra", "cleanup:" + findPro} {
		row := findRow(p, FindActions, id)
		if row == nil {
			t.Errorf("Find has no %s, want it among the actions", id)
			continue
		}
		if a := actions[row.Verb]; a.Confirm == nil {
			t.Errorf("%q runs without asking, want its question so the app draws its ellipsis", row.Label)
		}
	}
}
