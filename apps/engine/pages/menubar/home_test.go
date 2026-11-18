package menubar

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/awake"
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
	return Page(w.r, page.Context{Machine: heroPro, Kind: "computer"})
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
			if v.Text != "" {
				parts = append(parts, v.Text)
			}
		}
		out = append(out, strings.Join(parts, " | "))
	}
	return out
}

func tile(t *testing.T, p page.Page, caption string) page.Tile {
	t.Helper()
	for _, it := range homeSection(t, p, SectionTiles).Items {
		if it.Tile != nil && it.Tile.Caption == caption {
			return *it.Tile
		}
	}
	t.Fatalf("no tile %s", caption)
	return page.Tile{}
}

// invariant: SPEC A4 Home: under the hero, the tiles, the servers, the
// Macs, in that order.
func TestHomeListsItsTilesThenItsServersAndMacs(t *testing.T) {
	got := sectionIDs(homeOf(t, newHeroWorld()))
	if want := []string{SectionHero, SectionTiles, SectionServers, SectionMacs}; !reflect.DeepEqual(got[:4], want) {
		t.Fatalf("sections %v, want %v first", got, want)
	}
}

// invariant: SPEC A4 Home 4: a server's row says it is healthy, how long
// it is up and its ports; red while down, its count while it restarts.
// invariant: a server's ports are those reached beyond it: the tunnel
// connector's loopback listener inside it is none.
func TestAServerRowSaysItsHealth(t *testing.T) {
	w := newHeroWorld()
	infra := &w.mini().VMs[0]
	infra.IP, infra.SSH, infra.Guest = "192.168.64.7", true, "connected"
	w.r.Ports.Macs = []ports.MacPorts{{Name: heroMini, Ports: []ports.Port{
		{Machine: machine.Ref{Name: "infra", On: heroMini}, Port: 443, Reach: ports.Outside},
		{Machine: machine.Ref{Name: "infra", On: heroMini}, Port: 6443, Reach: ports.Tailnet},
		{Machine: machine.Ref{Name: "infra", On: heroMini}, Port: 20241, Process: "connector", Reach: ports.ThisMachine},
	}}}
	if got := rowWords(homeSection(t, homeOf(t, w), SectionServers)); !reflect.DeepEqual(got, []string{"infra | healthy | up 12 days | 2 ports"}) {
		t.Fatalf("healthy %q", got)
	}
	r := homeSection(t, homeOf(t, w.down()), SectionServers).Items[0].Row
	if r.Mark != page.Broken || r.Detail.Items[0].Text != "down since 14:28" || r.Detail.Items[0].Signal != page.SignalFailing {
		t.Fatalf("down %+v", r)
	}
	w.mini().VMs[0].Restart = &model.RestartWalk{Step: 2}
	r = homeSection(t, homeOf(t, w), SectionServers).Items[0].Row
	if r.Detail.Items[1].Text != "2 of 4" || r.Glyph.Progress != 250 {
		t.Fatalf("restarting %+v", r)
	}
}

// invariant: SPEC A4 Home 5: a Mac's row says how many VMs it runs, its
// memory free for a new VM and offers New VM there; an
// asleep one says when it was seen; one Mac alone adds Add a Mac.
func TestAMacRowSaysWhatRunsThere(t *testing.T) {
	w := newHeroWorld()
	air := model.MacView{Name: "macbook-air", Asleep: true, Answered: heroNow.Add(-2 * time.Hour), Laptop: true}
	w.r.All.Macs = append(w.r.All.Macs, air)
	s := homeSection(t, homeOf(t, w), SectionMacs)
	got := rowWords(s)
	if want := []string{"This Mac | 1 VM", "mac-mini | 1 VM", "macbook-air | asleep | seen 2 hr ago"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("macs %q, want %q", got, want)
	}
	mini := s.Items[1].Row
	if mini.Cells[0].Text != "60 GB free" || mini.Quick != ActionNewVMOn+heroMini || mini.Glyph.Ring != "cpu:"+heroMini {
		t.Fatalf("mini %+v", mini)
	}
	if asleep := s.Items[2].Row; asleep.Quick != "" || asleep.Signal != page.SignalAsleep || asleep.Glyph.Kind != "laptop" {
		t.Fatalf("asleep %+v", asleep)
	}
	w.r.All.Macs = w.r.All.Macs[:1]
	s = homeSection(t, homeOf(t, w), SectionMacs)
	if last := s.Items[len(s.Items)-1].Row; last.ID != RowAddMac || last.Verb != ActionAddMac {
		t.Fatalf("one Mac %+v", last)
	}
}

// invariant: SPEC A4 Home 2: the Ports tile counts what is shared, red
// what is exposed, never a loopback listener of a VM with its own device; Accounts what is signed out, else what is granted; Awake
// the hours left, or Off, its switch one click.
func TestTheTilesSayOneNumberEach(t *testing.T) {
	w := newHeroWorld().granted(t, "linux-1")
	w.r.Ports.Macs = []ports.MacPorts{{Name: heroPro, ThisMac: true, Shared: []ports.SharedPort{{VM: "linux-1", GuestPort: 5173}}, Ports: []ports.Port{
		{Machine: machine.Ref{Name: "infra", On: heroPro}, Port: 20241, Process: "connector", Reach: ports.ThisMachine, Device: true},
	}}}
	until := heroNow.Add(4*time.Hour + 20*time.Minute)
	w.r.Awake = &awake.Report{Mac: heroPro, Engine: awake.EngineRunning, Session: &awake.Session{Until: until}}
	p := homeOf(t, w)
	if got := tile(t, p, "Ports"); got.Number.N != 1 || got.Word != "shared" || got.Nav != PortsID {
		t.Fatalf("ports %+v", got)
	}
	if got := tile(t, p, "Accounts"); got.Number.N != 1 || got.Word != "granted" {
		t.Fatalf("accounts %+v", got)
	}
	got := tile(t, p, "Awake")
	toggle := actionOf(p, got.Toggle)
	if got.Number.Text != "4 hr" || got.Word != "left" || toggle.ID != ActionAwakeOff || !toggle.Picked {
		t.Fatalf("awake %+v, its switch %+v", got, toggle)
	}
	w.exposed()
	w.r.Awake.Session = nil
	p = homeOf(t, w)
	if shared := tile(t, p, "Ports"); shared.Number.N != 1 || shared.Word != "exposed" || shared.Signal != page.SignalFailing {
		t.Fatalf("exposed %+v", shared)
	}
	got = tile(t, p, "Awake")
	toggle = actionOf(p, got.Toggle)
	if got.Number.Text != "Off" || toggle.ID != ActionKeepAwake || toggle.Form != nil || toggle.Picked {
		t.Fatalf("off %+v, its switch %+v", got, toggle)
	}
}

// invariant: the Awake tile says how This Mac sleeps by awake.ModeOf's
// order: Never sleeps wins over a keep-awake.
func TestTheAwakeTileSaysAlwaysWhileTheMacNeverSleeps(t *testing.T) {
	w := newHeroWorld()
	w.r.Awake = &awake.Report{Now: heroNow, Mac: heroPro, Engine: awake.EngineRunning, NeverSleeps: true, NeverHeld: true, Session: &awake.Session{Until: heroNow.Add(2 * time.Hour)}}
	if got := tile(t, homeOf(t, w), "Awake"); got.Number.Text != "Always" || got.Word != "" {
		t.Fatalf("awake %+v", got)
	}
}
