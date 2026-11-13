package menubar

import (
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

var macNow = time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)

const (
	proMac  = "jterrazz-macbook-pro"
	miniMac = "jterrazz-mac-mini"
	airMac  = "jterrazz-macbook-air"
)

func macFleet() *model.AllMachines {
	server := model.VM{Name: "infra", Lifetime: model.AlwaysOn, State: model.Running, Want: "running", Resources: model.Resources{MemoryBytes: 24 << 30},
		BootAt: macNow.Add(-12 * 24 * time.Hour), CheckStatus: model.CheckStatus{State: model.CheckPassing}}
	waits := model.VM{Name: "review-bot", Lifetime: model.Persistent, State: model.Running, Resources: model.Resources{MemoryBytes: 4 << 30},
		Agent: &model.AgentView{State: "waiting", Task: "fixes the token refresh", Ask: "PR #42 is ready", Since: macNow.Add(-6 * time.Minute)}}
	works := model.VM{Name: "linux-3", Lifetime: model.Ephemeral, State: model.Running, Resources: model.Resources{MemoryBytes: 8 << 30},
		Agent: &model.AgentView{State: "working", Task: "refresh the auth token"}}
	return &model.AllMachines{Read: macNow, Macs: []model.MacView{
		{Name: proMac, User: "jterrazz", ThisMac: true, Reachable: true, Answered: macNow, Laptop: true, Hardware: coremachine.Hardware{Model: "Mac16,5", Chip: "Apple M4 Max", Memory: 48 << 30}},
		{Name: miniMac, User: "jterrazz", Reachable: true, Answered: macNow, Hardware: coremachine.Hardware{Model: "Mac16,11", Chip: "Apple M4 Pro", Memory: 64 << 30},
			VMs: []model.VM{works, waits, server}},
		{Name: airMac, User: "jterrazz", Answered: macNow.Add(-2 * time.Hour), Asleep: true, Laptop: true},
	}}
}

func macPageOf(t *testing.T, name string, edit func(*Machine)) page.Page {
	t.Helper()
	m := Machine{Name: name, All: macFleet(), Now: macNow, Zone: time.UTC, Asks: func(string, coremachine.Kind, ...string) string { return LevelYesNo },
		Summary: &overview.Summary{Head: overview.Head{Name: name, Kind: coremachine.Computer, Up: 21 * 24 * time.Hour}}}
	if edit != nil {
		edit(&m)
	}
	page.Strict(true)
	defer page.Strict(false)
	return MachinePage(m)
}

func sectionOf(p page.Page, id string) *page.Section {
	for i := range p.Sections {
		if p.Sections[i].ID == id {
			return &p.Sections[i]
		}
	}
	return nil
}

func labels(items []page.Item) []string {
	var out []string
	for _, it := range items {
		if it.Row != nil {
			out = append(out, it.Row.Label)
		}
	}
	return out
}

// invariant: SPEC A4, B3: a Mac's page carries every segment's sections at
// once, so the app picks a segment on its own and asks the engine nothing.
func TestAMacPageCarriesEverySegmentSoPickingOneAsksNothing(t *testing.T) {
	p := macPageOf(t, miniMac, nil)
	ids := make([]string, 0, len(p.Segments))
	for _, s := range p.Segments {
		ids = append(ids, s.ID)
	}
	if !slices.Equal(ids, []string{SegmentOverview, SegmentActivity, SegmentPorts, SegmentAccounts}) || !p.Segments[0].Picked {
		t.Fatalf("segments %q, want Overview picked then Activity, Ports, Accounts", ids)
	}
	for _, seg := range ids {
		if !slices.ContainsFunc(p.Sections, func(s page.Section) bool { return s.Segment == seg }) {
			t.Errorf("segment %s has no section in the page", seg)
		}
	}
	if s := sectionOf(p, SectionSummary); s == nil || s.Title != "mac-mini" || len(s.Caption) != 4 || s.Caption[0].Text != "Mac mini" || s.Caption[3].Text != "up 21 days" {
		t.Fatalf("facts %+v, want Mac mini, M4 Pro, 64 GB, up 21 days", s)
	}
	if p.Header.Glyph.Kind != "desktop" || !slices.Equal(p.Header.Buttons, []string{ActionNewVM, ActionMenu}) {
		t.Fatalf("header %+v, want the desktop glyph, New VM and the menu", p.Header)
	}
}

// invariant: SPEC A4: a Mac's Needs you is its own and its VMs', red
// first, three rows before the fold that says how many more.
func TestAMacsNeedsAreItsOwnRedFirstThreeBeforeTheFold(t *testing.T) {
	p := macPageOf(t, miniMac, func(m *Machine) {
		mini := &m.All.Macs[1]
		mini.VMs[2].State, mini.VMs[2].CheckStatus = model.Stopped, model.CheckStatus{State: model.CheckFailing, Since: macNow.Add(-4 * time.Minute)}
		m.Shares = &accounts.AllMachines{Now: macNow, Macs: []accounts.MacAccounts{{Name: miniMac, Shares: []accounts.Shared{
			{Account: "acme", VM: "linux-3", At: macNow.Add(-time.Hour), Until: macNow.Add(12 * time.Minute)},
			{Account: "globex", VM: "linux-3", At: macNow.Add(-time.Hour), Until: macNow.Add(12 * time.Minute)},
		}}}}
		m.Ports = &ports.AllMachines{Macs: []ports.MacPorts{{Name: proMac, ThisMac: true, Ports: []ports.Port{
			{Machine: coremachine.Ref{Name: proMac}, Port: 5000, Process: "ControlCenter", Reach: ports.Outside},
		}}}}
	})
	needs := sectionOf(p, SectionNeeds)
	if needs == nil {
		t.Fatal("no Needs you on a Mac whose server is down")
	}
	if got := labels(needs.Items); !slices.Equal(got, []string{"infra", "review-bot", "linux-3", "linux-3"}) {
		t.Fatalf("needs %q, want infra down first, then the wait, then the two shares ending; This Mac's port is not the mini's", got)
	}
	if needs.Top != 3 || needs.Rest == nil || needs.Rest.Row.Label != "1 more" || needs.Rest.Row.Detail.Text != "Show less" {
		t.Fatalf("fold top %d rest %+v, want three rows then 1 more", needs.Top, needs.Rest)
	}
	if needs.Caption[0].Text != "4" || needs.Caption[0].Signal != page.SignalFailing {
		t.Fatalf("count %+v, want 4 in red", needs.Caption)
	}
	vms := sectionOf(p, SectionVMs)
	if got := labels(vms.Items); !slices.Equal(got, []string{"infra", "linux-3", "review-bot"}) {
		t.Fatalf("VMs %q, want the server first", got)
	}
	if infra := vms.Items[0].Row; infra.Verb != "down:infra" || infra.Mark != page.Broken {
		t.Fatalf("infra's row %+v, want it red with Restart", infra)
	}
}

// invariant: SPEC A4: This Mac's VMs read the working one first with its
// agent's step, then the finished, then the stopped, and every row, the
// finished one too, opens its VM's page.
func TestAMacsVMsListTheWorkingFirstAndEachOpensItsPage(t *testing.T) {
	p := macPageOf(t, proMac, func(m *Machine) {
		m.All.Macs[0].VMs = []model.VM{
			{Name: "agent-7f", Lifetime: model.Ephemeral, State: model.Running, Agent: &model.AgentView{State: model.AgentFinished, Reason: "tests pass"}},
			{Name: "api", Lifetime: model.Persistent, State: model.Stopped},
			{Name: "linux-1", Lifetime: model.Ephemeral, State: model.Running,
				Agent: &model.AgentView{State: "working", Task: "builds the landing page", Step: []string{"running tests", "41/58 pass"}}},
		}
	})
	vms := sectionOf(p, SectionVMs)
	if got := labels(vms.Items); !slices.Equal(got, []string{"linux-1", "agent-7f", "api"}) {
		t.Fatalf("VMs %q, want the working linux-1, then the finished agent-7f, then the stopped api", got)
	}
	if d := words(vms.Items[0].Row.Detail); d != "running tests | 41/58 pass" {
		t.Errorf("linux-1 reads %q, want its agent's step as Home says it", d)
	}
	for _, it := range vms.Items {
		if want := machinePage(it.Row.Label); it.Row.Nav != want {
			t.Errorf("%s's row opens %q, want its page %q", it.Row.Label, it.Row.Nav, want)
		}
	}
	if finished := vms.Items[1].Row; finished.Verb != ActionCleanup {
		t.Errorf("agent-7f's row offers %q, want Clean up beside its way in", finished.Verb)
	}
}

// invariant: SPEC A4: an exposed port on This Mac leads its card, in its
// own words; its app is the one fact after them, and the first to go.
func TestAnExposedPortLeadsThisMacsNeeds(t *testing.T) {
	p := macPageOf(t, proMac, func(m *Machine) {
		m.Ports = &ports.AllMachines{Macs: []ports.MacPorts{{Name: proMac, ThisMac: true, Ports: []ports.Port{
			{Machine: coremachine.Ref{Name: proMac}, Port: 5000, Process: "ControlCenter", Reach: ports.Outside},
		}}}}
	})
	needs := sectionOf(p, SectionNeeds)
	if needs == nil || needs.Items[0].Row.Label != "Port 5000 exposed" || needs.Items[0].Row.Glyph.Kind != "symbol:exclamationmark.shield" {
		t.Fatalf("This Mac's needs %+v, want Port 5000 exposed first", needs)
	}
	if d := needs.Items[0].Row.Detail.Items; len(d) != 1 || d[0].Text != "AirPlay Receiver" || d[0].Pri == 0 {
		t.Fatalf("the exposed row's facts %+v, want only AirPlay Receiver, dropped first when the row is narrow: the page is This Mac", d)
	}
}

// invariant: SPEC A4: an asleep Mac has no segment and no New VM: it says
// it sleeps, when it was seen, and offers Wake and Tell me when it wakes.
func TestAnAsleepMacOffersWakeAndNoSegment(t *testing.T) {
	p := macPageOf(t, airMac, nil)
	if len(p.Segments) != 0 || slices.Contains(p.Header.Buttons, ActionNewVM) {
		t.Fatalf("an asleep Mac shows segments %v and buttons %v", p.Segments, p.Header.Buttons)
	}
	empty := sectionOf(p, SectionAsleep).Items[0].Empty
	if empty.Text != "macbook-air is asleep" || !slices.Equal(empty.Actions, []string{ActionWake, ActionRemind}) {
		t.Fatalf("asleep card %+v", empty)
	}
	facts := sectionOf(p, SectionSummary).Caption
	if facts[len(facts)-1].Text != "seen 2 hr ago" {
		t.Fatalf("facts %+v, want it seen 2 hr ago", facts)
	}
}

// invariant: SPEC A4, B5.15: a Mac not heard from for more than 30 s
// keeps its last values, marked as home marks them: seen when, checking,
// its gauges and rows stale.
func TestAMacNotHeardFromFor30sIsMarkedStale(t *testing.T) {
	silent := func(d time.Duration) page.Page {
		return macPageOf(t, miniMac, func(m *Machine) {
			m.All.Macs[1].Reachable, m.All.Macs[1].Answered = false, macNow.Add(-d)
		})
	}
	if p := silent(20 * time.Second); p.Header.Stale != nil {
		t.Fatalf("a Mac silent for 20 s is marked %+v", p.Header.Stale)
	}
	p := silent(40 * time.Second)
	if p.Header.Stale == nil || p.Header.Stale.Items[0].Text != "seen 40 s ago" || p.Header.Stale.Items[1].Text != "checking" {
		t.Fatalf("stale %+v, want seen 40 s ago, checking", p.Header.Stale)
	}
	for _, it := range sectionOf(p, SectionGauges).Items {
		if !it.Gauge.Stale {
			t.Errorf("gauge %s is not stale", it.Gauge.Title)
		}
	}
	for _, it := range sectionOf(p, SectionVMs).Items {
		if it.Row.Signal != page.SignalStale {
			t.Errorf("row %s is not stale", it.Row.ID)
		}
	}
}

// invariant: SPEC A4: a Mac with no VM says so in its VMs card, with the
// memory free for the first one and New VM there, not in its header.
func TestAMacWithNoVMOffersTheFirstInItsCard(t *testing.T) {
	p := macPageOf(t, proMac, nil)
	if slices.Contains(p.Header.Buttons, ActionNewVM) {
		t.Fatal("New VM sits in the header of a Mac with no VM")
	}
	empty := sectionOf(p, SectionVMs).Items[0].Empty
	if empty == nil || empty.Text != "No VMs on This Mac yet" || empty.Sub != "44 GB free for the first one" || !slices.Equal(empty.Actions, []string{ActionNewVM}) {
		t.Fatalf("empty card %+v", empty)
	}
}

// invariant: SPEC A4, B5.10: a VM being created shows its ring and its
// birth bar with its step; once ready it lands with its check.
func TestAVMBeingCreatedShowsItsBirthThenLands(t *testing.T) {
	born := func(edit func(*model.VM)) page.Row {
		vm := model.VM{Name: "linux-4", Lifetime: model.Ephemeral, State: model.Creating, Resources: model.Resources{MemoryBytes: 8 << 30}}
		edit(&vm)
		p := macPageOf(t, miniMac, func(m *Machine) { m.All.Macs[1].VMs = append(m.All.Macs[1].VMs, vm) })
		items := sectionOf(p, SectionVMs).Items
		return *items[len(items)-1].Row
	}
	creating := born(func(vm *model.VM) { vm.Birth = model.BirthAt(model.BornBooting, macNow) })
	if creating.Birth == nil || creating.Birth.N != 400 || creating.Glyph.Progress != 400 || words(creating.Detail) != "creating | booting" {
		t.Fatalf("creating row %+v", creating)
	}
	ready := born(func(vm *model.VM) { vm.State, vm.ReadyAt = model.Running, macNow.Add(-2*time.Second) })
	if !ready.Fresh || ready.Glyph.Kind != "symbol:checkmark.circle.fill" || ready.Quick != ActionAttach+"linux-4" {
		t.Fatalf("ready row %+v", ready)
	}
}

// invariant: SPEC A4's care card: Awake with its switch and what pauses,
// and the upkeep folded behind its worst item.
func TestTheCareCardSaysAwakeAndTheWorstUpkeep(t *testing.T) {
	p := macPageOf(t, miniMac, func(m *Machine) {
		m.Awake = &awake.Report{Mac: miniMac, Engine: awake.EngineRunning, NeverSleeps: true, NeverHeld: true, Servers: []string{"infra"}}
		m.Summary.Needs = []overview.Need{{Section: "settings", Subject: "firewall", Text: "Firewall: off"}, {Section: "apps", Subject: "updates", Text: "2 app updates", Call: CallUpdate}}
		m.Summary.Lines = append(m.Summary.Lines, overview.Line{Section: "disk", Parts: []overview.Part{{Bytes: 388_000_000_000, Word: "free"}, {Word: "can rebuild"}}})
		m.Samples = []activity.Reading{{Sample: activity.Sample{MemTotal: 64 << 30, MemUsed: 45 << 30}}}
	})
	care := sectionOf(p, SectionCare).Items
	if r := care[0].Row; r.Label != "Never sleeps" || r.Detail.Text != "hosts infra, which pauses if it sleeps" || r.Cells[0].Kind != page.KindOn || r.Verb == "" {
		t.Fatalf("awake row %+v", r)
	}
	head := care[1].Group.Head
	if words(head.Cells[0]) != "firewall off | 1 more" || care[1].Group.Fold != page.Folded || len(care[1].Group.Items) != 3 {
		t.Fatalf("upkeep %q folded %s over %d rows", words(head.Cells[0]), care[1].Group.Fold, len(care[1].Group.Items))
	}
	disk := false
	for _, it := range care[1].Group.Items {
		if it.Row.Label == "Apps" && words(it.Row.Detail) != "2 updates" {
			t.Fatalf("the Apps item reads %q, want its updates said once", words(it.Row.Detail))
		}
		if it.Row.Label == "Disk" {
			disk = true
			if d := it.Row.Detail; d.Kind != page.KindList || len(d.Items) != 1 || d.Items[0].Unit != "free" {
				t.Fatalf("the Disk item reads %+v, want its free space alone when nothing can be rebuilt", d)
			}
		}
	}
	if !disk {
		t.Fatal("the upkeep unfolds to no Disk item")
	}
	if g := sectionOf(p, SectionGauges).Items[1].Gauge; g.Value.Text != "19 GB free" || g.Bar == nil {
		t.Fatalf("memory gauge %+v, want 19 GB free over its bar", g)
	}
}

// invariant: SPEC A4: a Mac that never sleeps unfolds its upkeep to its
// readiness too, "6 of 6" when every item holds; one that does not is an
// open item the upkeep's head counts, and names itself.
func TestAMacThatNeverSleepsUnfoldsItsUpkeepToItsReadiness(t *testing.T) {
	ready := func(m *Machine) {
		m.Summary.Readiness = []overview.Check{
			{Name: model.ItemRestartsAfterPowerCut, State: catalog.StateOK}, {Name: model.ItemNeverSleeps, State: catalog.StateOK},
			{Name: model.ItemLogsIn, State: catalog.StateOK}, {Name: model.ItemDiskUnlock, State: catalog.StateOK},
			{Name: model.ItemPlannedRestarts, State: catalog.StateOK}, {Name: model.ItemEngineAtLogin, State: catalog.StateOK},
		}
	}
	upkeep := func(p page.Page) *page.Group {
		for _, it := range sectionOf(p, SectionCare).Items {
			if it.Group != nil && it.Group.Head.ID == RowUpkeep {
				return it.Group
			}
		}
		t.Fatal("no upkeep row")
		return nil
	}
	g := upkeep(macPageOf(t, miniMac, ready))
	last := g.Items[len(g.Items)-1].Row
	if words(g.Head.Cells[0]) != "all fine" || last.Label != "Ready to host servers" || words(last.Detail) != "6 of 6" || last.Mark != page.OK {
		t.Fatalf("upkeep %q unfolds to %q %q", words(g.Head.Cells[0]), last.Label, words(last.Detail))
	}
	g = upkeep(macPageOf(t, miniMac, func(m *Machine) {
		ready(m)
		m.Summary.Readiness[3].State = catalog.StateDegraded
	}))
	last = g.Items[len(g.Items)-1].Row
	if words(g.Head.Cells[0]) != "5 of 6 ready" || words(last.Detail) != "5 of 6 ready | disk unlock after a power cut" || last.Mark != page.Attention {
		t.Fatalf("upkeep %q unfolds to %q", words(g.Head.Cells[0]), words(last.Detail))
	}
	if g := upkeep(macPageOf(t, proMac, nil)); slices.ContainsFunc(g.Items, func(it page.Item) bool { return it.Row.ID == RowReady }) {
		t.Fatal("a Mac with no readiness unfolds to none")
	}
}
