package menubar

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

const (
	findPro  = "macbook-pro"
	findMini = "mac-mini"
	findAir  = "macbook-air"
	gib      = int64(1) << 30
)

var findNow = time.Date(2026, 11, 7, 15, 30, 0, 0, time.UTC)

func asksYesNo(gesture string, k machine.Kind, _ ...string) string {
	if gesture == "restart-server" && k == machine.Server {
		return LevelTypeName
	}
	return LevelNone
}

// value: the prototype's fleet: This Mac with linux-1 running and agent-7f
// stopped, mac-mini with infra healthy, linux-3 and review-bot running,
// macbook-air asleep; Vite of linux-1 shared, Postgres
// listening in linux-3; This Mac awake until 18:00 and a password of its
// vault that fits linux-1 and agent-7f.
func findFleet() FindReading {
	vm := func(name, mac string, life model.Lifetime, memory int64) model.VM {
		return model.VM{Name: name, Mac: mac, Lifetime: life, State: model.Running, Resources: model.Resources{MemoryBytes: memory * gib}}
	}
	stopped := vm("agent-7f", findPro, model.Ephemeral, 8)
	stopped.State = model.Stopped
	infra := vm("infra", findMini, model.AlwaysOn, 24)
	infra.Want, infra.Guest, infra.SSH, infra.IP = "running", "connected", true, "192.168.64.7"
	infra.CheckStatus = model.CheckStatus{State: model.CheckPassing}
	all := model.AllMachines{Macs: []model.MacView{
		{Name: findPro, User: "you", ThisMac: true, Reachable: true, Laptop: true, Hardware: machine.Hardware{Memory: 48 * gib}, VMs: []model.VM{
			vm("linux-1", findPro, model.Ephemeral, 8),
			stopped,
		}},
		{Name: findMini, User: "you", Reachable: true, Hardware: machine.Hardware{Memory: 64 * gib}, VMs: []model.VM{
			infra,
			vm("linux-3", findMini, model.Ephemeral, 8),
			vm("review-bot", findMini, model.Persistent, 4),
		}},
		{Name: findAir, User: "you", Laptop: true, Answered: findNow.Add(-2 * time.Hour)},
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
	entry := accounts.VaultEntry{ID: "pw-1", Label: "deploy token", Program: accounts.Programs()[0]}
	signIns := &SignIns{Vault: &accounts.VaultRead{Mac: findPro, Entries: []accounts.VaultEntry{entry}}, Offers: map[string]accounts.GrantOffer{"pw-1": offerFor("linux-1", "agent-7f")}}
	held := &awake.Report{Engine: awake.EngineRunning, Mac: findPro,
		Session: &awake.Session{Lid: true, StartedAt: findNow.Add(-time.Hour), Until: time.Date(2026, 11, 7, 18, 0, 0, 0, time.UTC)}}
	return FindReading{Reading: Reading{All: &all, Ports: &listed, SignIns: signIns, Awake: held}, Now: findNow, Zone: time.UTC, Asks: asksYesNo}
}

// value: the same fleet once infra fell at 15:10.
func findFleetDown() FindReading {
	r := findFleet()
	infra := &r.All.Macs[1].VMs[0]
	infra.State, infra.Cause = model.Stopped, "out of memory"
	infra.Restarts = model.Restarts{Count: 5, Streak: 5, Looping: true}
	infra.CheckStatus = model.CheckStatus{State: model.CheckFailing, Since: time.Date(2026, 11, 7, 15, 10, 0, 0, time.UTC)}
	standUnsent(r.All)
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
	p := Find(findFleet(), page.Context{Machine: findPro, Kind: "computer"})
	if err := page.Check(p); err != nil {
		t.Fatalf("the Find index fails its checks: %v", err)
	}

	cases := []struct {
		section string
		want    []findSaid
	}{
		{FindNeeds, nil},
		{FindMachines, []findSaid{
			{"This Mac", "2 VMs | 36 GB free", "Open", "cmd+1", ""},
			{"mac-mini", "3 VMs | 24 GB free", "Open", "cmd+2", ""},
			{"macbook-air", "asleep | seen 2 hr ago", "Open", "cmd+3", ""},
			{"agent-7f", "stopped | This Mac", "Open", "", ""},
			{"linux-1", "running | This Mac", "Open", "", "Open in Terminal"},
			{"infra", "healthy | mac-mini", "Open", "", ""},
			{"linux-3", "running | mac-mini", "Open", "", "Open in Terminal"},
			{"review-bot", "running | mac-mini", "Open", "", "Open in Terminal"},
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
			{"Needs you", "", "Open", "", ""},
			{"Ports", "All machines", "Open", "cmd+p", ""},
			{"Accounts", "All machines", "Open", "cmd+u", ""},
			{"Awake", "All Macs", "Open", "cmd+e", ""},
		}},
		{FindActions, []findSaid{
			{"Keep This Mac awake 4 h", "", "Keep awake", "", ""},
			{"New VM on This Mac", "36 GB free", "New VM", "cmd+n", ""},
			{"New VM on mac-mini", "24 GB free", "New VM", "", ""},
			{"Restart infra", "asks first", "Restart", "", ""},
			{"Clean up ephemeral workspaces on This Mac", "", "Clean up", "", ""},
		}},
	}
	for _, c := range cases {
		if got := entries(t, p, c.section); !slices.Equal(got, c.want) {
			t.Errorf("%s:\n got %+v\nwant %+v", c.section, got, c.want)
		}
	}
	if got, want := entries(t, p, FindVerbs), []findSaid{
		{"Open linux-1 in Terminal", "", "Open in Terminal", "cmd+enter", ""},
		{"Share a port of linux-1", "", "Share", "", ""},
		{"Grant a password to linux-1", "", "Run", "", ""},
		{"Share a port of infra", "", "Share", "", ""},
		{"Restart infra", "", "Restart", "", ""},
		{"Open linux-3 in Terminal", "", "Open in Terminal", "cmd+enter", ""},
		{"Share a port of linux-3", "", "Share", "", ""},
		{"Open review-bot in Terminal", "", "Open in Terminal", "cmd+enter", ""},
		{"Share a port of review-bot", "", "Share", "", ""},
	}; !slices.Equal(got, want) {
		t.Errorf("the VMs' verbs:\n got %+v\nwant %+v", got, want)
	}
	if r := findRow(p, FindMachines, "linux-1"); r == nil || r.Nav != "menubar.machine:linux-1" || !strings.Contains(words(r.Cells[0]), "workspace") {
		t.Errorf("linux-1's entry %+v, want its page and the tag workspace", r)
	}
	if r := findRow(p, FindPorts, "listening:linux-3:5432"); r == nil || !strings.Contains(words(r.Cells[0]), "| 5432 |") {
		t.Errorf("Postgres's entry %+v, want it found by its number", r)
	}
}

// invariant: SPEC A6 [find.down]: while infra is down Find leads with
// Restart infra, run once its name is typed, and says since when it is down.
func TestFindLeadsWithRestartInfraWhileItIsDown(t *testing.T) {
	p := Find(findFleetDown(), page.Context{Machine: findPro, Kind: "computer"})
	if err := page.Check(p); err != nil {
		t.Fatalf("the Find index fails its checks: %v", err)
	}
	needs := entries(t, p, FindNeeds)
	if len(needs) != 1 || needs[0] != (findSaid{"Restart infra", "down since 15:10", "Restart", "", ""}) {
		t.Errorf("Find's needs %+v, want Restart infra alone", needs)
	}
	restart := findRow(p, FindNeeds, "down:infra")
	if restart == nil || restart.Mark != page.Broken {
		t.Errorf("infra's need %+v, want the red mark of a broken machine", restart)
	}
	for _, a := range p.Actions {
		if restart != nil && a.ID == restart.Verb && (a.Confirm == nil || a.Confirm.Type != "infra" || a.Run == nil || a.Run.Call != CallRestartVM) {
			t.Errorf("Restart infra from Find runs %+v, want machines.restart once infra is typed", a)
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
		if action != nil && a.ID == action.Verb && (a.Confirm == nil || a.Confirm.Type != "infra" || a.Run == nil || a.Run.Call != CallRestartVM) {
			t.Errorf("Restart infra among the actions runs %+v, want machines.restart once infra is typed", a)
		}
	}
}

// invariant: a keep-awake from Find never shortens the term held: past
// 4 h left it extends the end by 4 h.
func TestFindExtendsAwakeWhenMoreThanFourHoursAreLeft(t *testing.T) {
	r := findFleet()
	r.Awake.Session.Until = time.Date(2026, 11, 7, 21, 0, 0, 0, time.UTC)
	p := Find(r, page.Context{Machine: findPro, Kind: "computer"})
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
	p := Find(r, page.Context{Machine: findPro, Kind: "computer"})
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

// invariant: an asleep Mac offers no clean-up, stopped ephemeral workspaces or not.
func TestFindOffersNoCleanUpOnAnAsleepMac(t *testing.T) {
	r := findFleet()
	r.All.Macs[2].VMs = []model.VM{{Name: "agent-9a", Mac: findAir, Lifetime: model.Ephemeral, State: model.Stopped}}
	p := Find(r, page.Context{Machine: findPro, Kind: "computer"})
	if row := findRow(p, FindActions, "cleanup:"+findAir); row != nil {
		t.Errorf("the asleep macbook-air offers %+v, want no clean-up", row)
	}
	if row := findRow(p, FindActions, "cleanup:"+findPro); row == nil {
		t.Errorf("This Mac lost its clean-up, want Clean up ephemeral workspaces on This Mac")
	}
}

// invariant: an action that asks first carries its question, which the app
// draws as the label's ellipsis: "Restart infra…" and "Clean up ephemeral workspaces
// on This Mac…" in the prototype.
func TestFindsActionsThatAskFirstCarryTheirQuestion(t *testing.T) {
	p := Find(findFleet(), page.Context{Machine: findPro, Kind: "computer"})
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

// invariant: SPEC A6 [find.awake]: every keep-awake Find offers ends at a
// time, a length or an hour.
func TestFindKeepsThisMacAwakeByATimeOnly(t *testing.T) {
	for name, r := range map[string]FindReading{"a session": findFleet(), "no session": func() FindReading {
		r := findFleet()
		r.Awake.Session = nil
		return r
	}()} {
		keeps := 0
		for _, a := range Find(r, page.Context{Machine: findPro, Kind: "computer"}).Actions {
			if a.Run == nil || a.Run.Call != CallKeepAwake {
				continue
			}
			keeps++
			for key := range a.Run.In {
				if !slices.Contains([]string{"For", "Until", "Lid", "Battery", "By"}, key) {
					t.Errorf("%s: Find's %s keeps awake by %q, which is no time", name, a.ID, key)
				}
			}
			if a.Run.In["For"] == "" && a.Run.In["Until"] == "" {
				t.Errorf("%s: Find's %s keeps awake with no end: %+v", name, a.ID, a.Run.In)
			}
		}
		if keeps != 1 {
			t.Errorf("%s: Find offers %d keep-awakes, want the one 4 h or its extension", name, keeps)
		}
	}
}

// invariant: a listener reached from outside keeps its address, then says
// it is public.
func TestFindSaysAPublicListenersAddressThenPublic(t *testing.T) {
	r := findFleet()
	r.Ports.Macs[1].Ports = append(r.Ports.Macs[1].Ports, ports.Port{Machine: machine.Ref{Name: "infra", On: findMini}, Port: 443, Process: "nginx", Reach: ports.Outside})
	row := findRow(Find(r, page.Context{Machine: findPro, Kind: "computer"}), FindPorts, "listening:infra:443")
	if row == nil || words(row.Detail) != "infra:443 | public" {
		t.Errorf("infra's public listener %+v, want infra:443, then public", row)
	}
}

// invariant: a listener bound to its VM's loopback is reached from that VM
// only: Find offers no Share for it.
func TestFindOffersNoShareOfAListenerOnItsVMsLoopback(t *testing.T) {
	r := findFleet()
	r.Ports.Macs[1].Ports = append(r.Ports.Macs[1].Ports, ports.Port{Machine: machine.Ref{Name: "linux-3", On: findMini}, Port: 20241, Process: "connector", Reach: ports.ThisMachine})
	if row := findRow(Find(r, page.Context{Machine: findPro, Kind: "computer"}), FindPorts, "listening:linux-3:20241"); row != nil {
		t.Errorf("linux-3's loopback listener %+v, want no entry", row)
	}
}

// invariant: SPEC 2-9: Needs you is a section of Find, so every need and
// the upkeep are reached when the hero lists the needs whole.
func TestFindOpensNeedsYou(t *testing.T) {
	row := findRow(Find(findFleet(), page.Context{Machine: findPro, Kind: "computer"}), FindSections, "needs")
	if row == nil || row.Nav != NeedsID {
		t.Errorf("the Needs you entry %+v, want it to open %s", row, NeedsID)
	}
}
