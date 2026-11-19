package menubar

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

func homeOf(t *testing.T, w *heroWorld) page.Page {
	t.Helper()
	for i := range w.r.All.Macs {
		w.r.All.Macs[i].User = "you"
	}
	judged := model.Judged(*w.r.All)
	r := w.r
	r.All = &judged
	return Page(r, page.Context{Machine: heroPro, Kind: "computer"})
}

func homeSection(t *testing.T, p page.Page, id string) page.Section {
	t.Helper()
	for _, s := range p.Sections {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("no section %s in %v", id, sectionIDs(p))
	return page.Section{}
}

func sectionIDs(p page.Page) []string {
	ids := make([]string, 0, len(p.Sections))
	for _, s := range p.Sections {
		ids = append(ids, s.ID)
	}
	return ids
}

func actionOf(p page.Page, id string) page.Action {
	for _, a := range p.Actions {
		if a.ID == id {
			return a
		}
	}
	return page.Action{}
}

func rowWords(s page.Section) []string {
	var out []string
	for _, it := range s.Items {
		if it.Row == nil {
			continue
		}
		parts := []string{it.Row.Label}
		for _, v := range append([]page.Value{it.Row.Detail}, it.Row.Detail.Items...) {
			switch {
			case v.Text != "":
				parts = append(parts, v.Text)
			case v.Label != "":
				parts = append(parts, v.Label)
			}
		}
		out = append(out, strings.Join(parts, " | "))
	}
	return out
}

func homeRow(t *testing.T, s page.Section, id string) page.Row {
	t.Helper()
	for _, it := range s.Items {
		if it.Row != nil && it.Row.ID == id {
			return *it.Row
		}
	}
	t.Fatalf("no row %s in %s", id, s.ID)
	return page.Row{}
}

// value: a world whose Macs were each read twice, their running VMs in
// each reading.
func (w *heroWorld) read() *heroWorld {
	w.r.Samples = map[string][]activity.Reading{}
	for _, m := range w.r.All.Macs {
		rs := make([]activity.Reading, 0, 3)
		for i := range 3 {
			r := activity.Reading{Sample: activity.Sample{At: heroNow.Add(time.Duration(i-3) * 2 * time.Second), CPUBusy: uint64(i * 300), CPUTotal: uint64(i * 1000),
				MemUsed: 8 << 30, MemTotal: 16 << 30, DiskTotal: 1000, DiskFree: 170}}
			for _, vm := range m.VMs {
				r.VMs = append(r.VMs, activity.VMUsage{VM: vm.Name, Usage: activity.Usage{CPU: float64(10 * (i + 1)), Memory: 5 << 30}})
			}
			rs = append(rs, r)
		}
		w.r.Samples[m.Name] = rs
	}
	return w
}

// invariant: Home is one kind per section, in one order: Needs you only
// when something needs you, This Mac's head, the servers, the VMs, the
// Macs, then the Keep awake, Ports and Accounts rows; its header is titled
// Station and carries the find field's prompt.
func TestHomeListsOneKindPerSectionUnderItsTitleAndSearch(t *testing.T) {
	p := homeOf(t, newHeroWorld())
	if got, want := sectionIDs(p), []string{SectionThisMac, SectionServers, SectionVMs, SectionMacs, SectionShortcuts}; !reflect.DeepEqual(got[:5], want) {
		t.Fatalf("sections %v, want %v first", got, want)
	}
	if h := p.Header; h == nil || h.Title != "Station" || h.Search != searchPrompt || !slices.Equal(h.Buttons, []string{ActionNewVM}) {
		t.Fatalf("header %+v, want Station, the prompt and New VM", h)
	}
	p = homeOf(t, newHeroWorld().exposed())
	if got := sectionIDs(p)[:2]; !reflect.DeepEqual(got, []string{SectionNeedsRows, SectionThisMac}) {
		t.Fatalf("sections %v, want Needs you first while a port is exposed", got)
	}
	if got := rowWords(homeSection(t, p, SectionNeedsRows)); !reflect.DeepEqual(got, []string{"Port 5000 | " + catalog.ReachableOutside}) {
		t.Fatalf("needs %q", got)
	}
	need := homeSection(t, p, SectionNeedsRows).Items[0].Row
	if actionOf(p, need.Verb).Label != "Review" {
		t.Fatalf("the need's verb %q, want Review", need.Verb)
	}
	if g := need.Glyph; g.Dot != page.DotFill || g.Signal != page.SignalFailing || g.Kind != "" {
		t.Fatalf("the need leads with %+v, want its signal's dot", g)
	}
}

// invariant: a server down, then a port exposed, each its own row ranked,
// each its verb.
func TestNeedsYouListsEveryNeedRankedEachWithItsVerb(t *testing.T) {
	p := homeOf(t, newHeroWorld().down().exposed().exposedOnMini(8080))
	s := homeSection(t, p, SectionNeedsRows)
	if got := rowWords(s); len(got) != 3 || !strings.HasPrefix(got[0], "infra") {
		t.Fatalf("needs %q, want infra first of 3", got)
	}
	for _, it := range s.Items {
		if it.Row.Verb == "" || actionOf(p, it.Row.Verb).ID == "" {
			t.Errorf("%s: its verb %q is no action", it.Row.Label, it.Row.Verb)
		}
	}
}

// invariant: a restart begins while the server is still down: from its
// first step it is its server's row, never a need nor counted.
func TestAServerRestartingFromDownIsNeitherANeedNorCounted(t *testing.T) {
	w := newHeroWorld().down()
	w.mini().VMs[0].Restart = &model.RestartWalk{Step: 2, At: heroNow}
	p := homeOf(t, w)
	if ids := sectionIDs(p); slices.Contains(ids, SectionNeedsRows) {
		t.Fatalf("sections %v hold Needs you", ids)
	}
	if b := p.Badge; b == nil || !b.Restarting || b.Down {
		t.Fatalf("badge %+v", p.Badge)
	}
	r := homeRow(t, homeSection(t, p, SectionServers), "infra")
	if got := rowWords(page.Section{Items: []page.Item{{Row: &r}}}); !reflect.DeepEqual(got, []string{"infra | restarting | 2 of 4"}) || r.Glyph.Progress != 250 {
		t.Fatalf("restarting %q %+v", got, r.Glyph)
	}
}

// invariant: a server is the kind that pages you: a fleet whose one server
// is down keeps the Servers section of that one row, before the VMs.
func TestServersKeepsItsSectionForItsOneRow(t *testing.T) {
	p := homeOf(t, newHeroWorld().down())
	s := homeSection(t, p, SectionServers)
	if got := rowWords(s); !reflect.DeepEqual(got, []string{"infra | down since 14:28"}) {
		t.Fatalf("servers %q", got)
	}
	if r := s.Items[0].Row; r.Mark != page.Broken || r.Mac != heroMini || r.Glyph.Signal != page.SignalFailing || r.Detail.Items[0].Signal != page.SignalFailing {
		t.Fatalf("down %+v", r)
	}
	if ids := sectionIDs(p); slices.Index(ids, SectionServers) > slices.Index(ids, SectionVMs) {
		t.Fatalf("sections %v, want Servers before VMs", ids)
	}
}

// invariant: a running VM's row says its standing and the memory its
// process holds, and ends in its CPU, a spark from its Mac's own
// readings, far Macs' too; every other standing says its lifetime and
// lists no key.
func TestARunningRowEndsInItsCPUSparkAndAStoppedOneInNone(t *testing.T) {
	w := newHeroWorld()
	w.r.All.Macs[0].VMs = append(w.r.All.Macs[0].VMs, model.VM{Name: "api", Lifetime: model.Persistent, State: model.Stopped})
	p := homeOf(t, w.read())
	vms := homeSection(t, p, SectionVMs)
	if got := rowWords(vms); !reflect.DeepEqual(got, []string{"linux-1 | running", "api | stopped | Persistent"}) {
		t.Fatalf("vms %q", got)
	}
	if vms.Caption[0].Text != "2" {
		t.Fatalf("the VMs head counts %+v", vms.Caption)
	}
	for _, c := range []struct{ vm, mac string }{{"linux-1", heroPro}, {"infra", heroMini}} {
		s := homeSection(t, p, SectionVMs)
		if c.vm == "infra" {
			s = homeSection(t, p, SectionServers)
		}
		r := homeRow(t, s, c.vm)
		key := "cpuh:" + c.vm
		want := LiveValue(key, w.r.Samples[c.mac])
		want.Live = key
		if len(r.Cells) != 1 || r.Cells[0].Kind != page.KindSpark || !reflect.DeepEqual(r.Cells[0], want) {
			t.Errorf("%s: its cell %+v, want %+v from %s's readings", c.vm, r.Cells, want, c.mac)
		}
		if mem := r.Detail.Items[1]; mem.Live != "memh:"+c.vm || mem.Kind != page.KindSize || mem.N != 5<<30 {
			t.Errorf("%s: its memory %+v", c.vm, mem)
		}
		if !slices.Contains(p.Live, key) || !slices.Contains(p.Live, "memh:"+c.vm) {
			t.Errorf("%s: the page follows %v", c.vm, p.Live)
		}
	}
	stopped := homeRow(t, vms, "api")
	if len(stopped.Cells) != 0 || stopped.Mark != page.Absent || stopped.Glyph.Dot != page.DotRing {
		t.Fatalf("stopped %+v", stopped)
	}
	for _, k := range p.Live {
		if strings.HasSuffix(k, ":api") {
			t.Fatalf("the page follows %s for a stopped VM", k)
		}
	}
}

// invariant: one rule for every online Mac's line: online, how many VMs
// it holds, its disk used, the shortest part dropped first; its CPU a
// spark; an asleep Mac says when it was seen and lists no key.
func TestAMacRowSaysOnlineItsVMsAndItsDiskAndAnAsleepOneNoKey(t *testing.T) {
	w := newHeroWorld()
	w.r.All.Macs = append(w.r.All.Macs, model.MacView{Name: "macbook-air", Asleep: true, Answered: heroNow.Add(-2 * time.Hour), Laptop: true})
	p := homeOf(t, w.read())
	s := homeSection(t, p, SectionMacs)
	if got, want := rowWords(s), []string{"This Mac | online | 1 VM | Disk", "mac-mini | online | 1 VM | Disk", "macbook-air | asleep | seen 2 hr ago"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("macs %q, want %q", got, want)
	}
	mini := homeRow(t, s, heroMini)
	vms, disk := mini.Detail.Items[1], mini.Detail.Items[2]
	if disk.Live != "disk:"+heroMini || disk.Kind != page.KindSpark || disk.Items[0].N != 830 || vms.Pri <= disk.Pri {
		t.Fatalf("mini's line %+v, want its disk 83 %% live, the VMs dropped first", mini.Detail.Items)
	}
	if c := mini.Cells; len(c) != 1 || c[0].Live != "cpu:"+heroMini || c[0].Kind != page.KindSpark || mini.Quick != ActionNewVMOn+heroMini {
		t.Fatalf("mini %+v", mini)
	}
	air := homeRow(t, s, "macbook-air")
	if len(air.Cells) != 0 || air.Quick != "" || air.Signal != page.SignalAsleep || air.Glyph.Ring != "" {
		t.Fatalf("asleep %+v", air)
	}
	for _, k := range p.Live {
		if strings.HasSuffix(k, ":macbook-air") {
			t.Fatalf("the page follows %s for an asleep Mac", k)
		}
	}
	w.r.All.Macs = w.r.All.Macs[:1]
	s = homeSection(t, homeOf(t, w), SectionMacs)
	if last := s.Items[len(s.Items)-1].Row; last.ID != RowAddMac || last.Verb != ActionAddMac {
		t.Fatalf("one Mac %+v", last)
	}
}

// invariant: This Mac's head carries the memory a new VM finds there.
func TestThisMacsHeadSaysItsFreeMemory(t *testing.T) {
	s := homeSection(t, homeOf(t, newHeroWorld()), SectionThisMac)
	if s.Title != page.ThisMac || len(s.Caption) != 1 || s.Caption[0].Text != "44 GB free" {
		t.Fatalf("this mac %+v", s)
	}
}

// invariant: the rows under the machines: Ports counts what is shared,
// red what is exposed, never a loopback listener of a VM with its own
// device; Accounts what is signed out, else what is granted; Keep awake
// its term, or off, its switch one click.
func TestTheShortcutRowsSayOneValueEach(t *testing.T) {
	w := newHeroWorld().granted(t, "linux-1")
	w.r.Ports.Macs = []ports.MacPorts{{Name: heroPro, ThisMac: true, Shared: []ports.SharedPort{{VM: "linux-1", GuestPort: 5173}}, Ports: []ports.Port{
		{Machine: machine.Ref{Name: "infra", On: heroPro}, Port: 20241, Process: "connector", Reach: ports.ThisMachine, Device: true},
	}}}
	w.r.Awake = &awake.Report{Mac: heroPro, Engine: awake.EngineRunning, Zone: time.UTC, Session: &awake.Session{Until: time.Date(2026, 11, 7, 18, 0, 0, 0, time.UTC)}}
	p := homeOf(t, w)
	s := homeSection(t, p, SectionShortcuts)
	if got := rowWords(s); !reflect.DeepEqual(got, []string{"Keep awake", "Ports", "Accounts"}) {
		t.Fatalf("shortcuts %q", got)
	}
	cell := func(id string) page.Value { return homeRow(t, s, id).Cells[0] }
	if cell(RowPorts).Text != "1 shared" || homeRow(t, s, RowPorts).Nav != PortsID || cell(RowAccounts).Text != "1 granted" || cell(RowAwake).Text != "until 18:00" {
		t.Fatalf("cells %+v %+v %+v", cell(RowPorts), cell(RowAccounts), cell(RowAwake))
	}
	if toggle := actionOf(p, homeRow(t, s, RowAwake).Verb); toggle.ID != ActionAwakeOff || !toggle.Picked {
		t.Fatalf("its switch %+v", toggle)
	}
	w.exposed()
	w.r.Awake.Session = nil
	p = homeOf(t, w)
	s = homeSection(t, p, SectionShortcuts)
	if c := cell(RowPorts); c.Text != "1 exposed" || c.Signal != page.SignalFailing {
		t.Fatalf("exposed %+v", c)
	}
	if toggle := actionOf(p, homeRow(t, s, RowAwake).Verb); cell(RowAwake).Text != "off" || toggle.ID != ActionKeepAwake || toggle.Form != nil || toggle.Picked {
		t.Fatalf("off %+v, its switch %+v", cell(RowAwake), toggle)
	}
}

// invariant: an empty fleet keeps the empty state, with the free memory
// of This Mac and New VM, and no + in the header.
func TestAnEmptyFleetIsTheEmptyStateWithNewVM(t *testing.T) {
	w := newHeroWorld()
	w.r.All.Macs = w.r.All.Macs[:1]
	w.r.All.Macs[0].VMs = nil
	p := homeOf(t, w)
	e := homeSection(t, p, SectionEmpty).Items[0].Empty
	if e == nil || e.Text != "No VMs yet" || e.Sub != "44 GB free on This Mac" || len(e.Actions) != 1 || e.Actions[0] != ActionNewVM {
		t.Fatalf("empty state %+v", e)
	}
	if len(p.Header.Buttons) != 0 {
		t.Fatalf("the empty fleet's header shows %v", p.Header.Buttons)
	}
}

// invariant: SPEC B5.15: a Mac silent for more than 30 s is checked: the
// header says so, its needs are drawn stale, an asleep Mac is not checked.
func TestAMacStillAnsweringIsCheckedAndItsNeedsDrawnStale(t *testing.T) {
	w := newHeroWorld().exposed().exposedOnMini(8080)
	w.mini().Reachable, w.mini().Answered = false, heroNow.Add(-40*time.Second)
	p := homeOf(t, w)
	if p.Header.Stale == nil || p.Header.Stale.Text != "checking mac-mini" {
		t.Fatalf("header %+v", p.Header)
	}
	signals := map[string]page.Signal{}
	for _, it := range homeSection(t, p, SectionNeedsRows).Items {
		signals[it.Row.Mac+" "+it.Row.Label] = it.Row.Signal
	}
	if signals["mac-mini Port 8080"] != page.SignalStale || signals["This Mac Port 5000"] != "" {
		t.Fatalf("signals %v", signals)
	}
	w.mini().Asleep = true
	if p = homeOf(t, w); p.Header.Stale != nil {
		t.Fatalf("an asleep Mac is checked: %v", p.Header.Stale)
	}
}
