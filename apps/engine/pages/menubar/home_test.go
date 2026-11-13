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
		w.r.All.Macs[i].User = "jterrazz"
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

func agent(name, state, task string) model.VM {
	return model.VM{Name: name, Lifetime: model.Ephemeral, State: model.Running, Agent: &model.AgentView{State: state, Task: task}}
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

// invariant: SPEC A4 Home: under the hero, the tiles, the agents, the
// servers, the Macs, in that order.
func TestHomeListsItsTilesThenItsAgentsServersAndMacs(t *testing.T) {
	got := sectionIDs(homeOf(t, newHeroWorld()))
	if want := []string{SectionHero, SectionTiles, SectionAgents, SectionServers, SectionMacs}; !reflect.DeepEqual(got[:5], want) {
		t.Fatalf("sections %v, want %v first", got, want)
	}
}

// invariant: SPEC A4 Home 3: past three live agents two stay, then the
// finished one, the others folded behind the line that names them.
func TestPastThreeLiveAgentsTwoStayAndTheRestFold(t *testing.T) {
	w := newHeroWorld().waiting("review-bot", "PR #42 is ready", 6*time.Minute)
	done := agent("agent-7f", model.AgentFinished, "the changelog")
	done.Agent.Reason = "tests pass"
	w.r.All.Macs[0].VMs = append(w.r.All.Macs[0].VMs, done)
	w.mini().VMs = append(w.mini().VMs, agent("linux-3", "working", "the auth token"), agent("linux-5", "working", "the db schema"))
	s := homeSection(t, homeOf(t, w), SectionAgents)
	ids := make([]string, 0, len(s.Items))
	for _, it := range s.Items {
		ids = append(ids, it.Row.ID)
	}
	if want := []string{"review-bot", "linux-1", "agent-7f", "linux-3", "linux-5"}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("agents %v, want %v", ids, want)
	}
	if s.Top != 3 || s.Rest == nil || s.Rest.Row.Label != "2 more" || s.Rest.Row.Cells[0].Text != "linux-3, linux-5" || s.Rest.Row.Detail.Text != "Show less" {
		t.Fatalf("fold top %d rest %+v", s.Top, s.Rest)
	}
	if got := s.Caption[0].Text; got != "3 working" {
		t.Fatalf("caption %q", got)
	}
}

// invariant: three live agents or fewer never fold, the finished last.
func TestThreeLiveAgentsShowWhole(t *testing.T) {
	w := newHeroWorld().waiting("review-bot", "PR #42 is ready", 6*time.Minute)
	w.r.All.Macs[0].VMs = append([]model.VM{agent("agent-7f", model.AgentFinished, "")}, w.r.All.Macs[0].VMs...)
	w.mini().VMs = append(w.mini().VMs, agent("linux-3", "working", "the auth token"))
	s := homeSection(t, homeOf(t, w), SectionAgents)
	if s.Rest != nil || s.Top != 0 {
		t.Fatalf("folded %d %+v", s.Top, s.Rest)
	}
	got := rowWords(s)
	if want := []string{"review-bot | waits for you", "linux-1 | working", "linux-3 | the auth token", "agent-7f | done"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("rows %q, want %q", got, want)
	}
}

// invariant: a working agent says its current step and its VM's CPU; a
// waiting one only its task; a starting one its ring and its task; a
// finished one its end, its Clean up, no page and no terminal.
func TestAnAgentRowSaysWhatItsAgentDoes(t *testing.T) {
	w := newHeroWorld().waiting("review-bot", "PR #42 is ready", time.Minute)
	w.r.All.Macs[0].VMs[0].Agent.Step = []string{"running tests", "41/58 pass"}
	born := agent("linux-4", "working", "fix the flaky test")
	born.State, born.Birth = model.Creating, &model.Birth{Progress: 400}
	done := agent("agent-7f", model.AgentFinished, "")
	done.Agent.Reason = "tests pass"
	w.mini().VMs = append(w.mini().VMs, born)
	w.r.All.Macs[0].VMs = append(w.r.All.Macs[0].VMs, done)
	rows := map[string]page.Row{}
	for _, it := range homeSection(t, homeOf(t, w), SectionAgents).Items {
		rows[it.Row.ID] = *it.Row
	}
	if r := rows["linux-1"]; r.Trail != "cpu" || r.Cells[0].Live != "cpuh:linux-1" || r.Detail.Items[0].Text != "running tests" || r.Mac != page.ThisMac || r.Quick != ActionAttach+"linux-1" {
		t.Fatalf("working %+v", r)
	}
	if r := rows["review-bot"]; r.Trail != "" || r.Glyph.Signal != page.SignalWaits || r.Mac != "mac-mini" {
		t.Fatalf("waiting %+v", r)
	}
	if r := rows["linux-4"]; r.Glyph.Progress != 400 || r.Detail.Items[0].Text != "starting" || r.Detail.Items[1].Text != "fix the flaky test" {
		t.Fatalf("starting %+v", r)
	}
	if r := rows["agent-7f"]; r.Detail.Text != "tests pass" || r.Verb != ActionCleanup+":"+heroPro || r.Nav != "" || r.Quick != "" {
		t.Fatalf("finished %+v", r)
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

// invariant: SPEC A4 Home 5: a Mac's row says its agents at work and
// waiting, its memory free for a new VM and offers New VM there; an
// asleep one says when it was seen; one Mac alone adds Add a Mac.
func TestAMacRowSaysWhatRunsThere(t *testing.T) {
	w := newHeroWorld().waiting("review-bot", "PR #42 is ready", time.Minute)
	air := model.MacView{Name: "jterrazz-macbook-air", Asleep: true, Answered: heroNow.Add(-2 * time.Hour), Laptop: true}
	w.r.All.Macs = append(w.r.All.Macs, air)
	s := homeSection(t, homeOf(t, w), SectionMacs)
	got := rowWords(s)
	if want := []string{"This Mac | 1 working", "mac-mini | 1 waiting", "macbook-air | asleep | seen 2 hr ago"}; !reflect.DeepEqual(got, want) {
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
// what is exposed, never a loopback listener of a VM with its own device; Accounts what is signed out, else what is shared; Awake
// the hours left, or Off, its switch one click.
func TestTheTilesSayOneNumberEach(t *testing.T) {
	w := newHeroWorld().lease("linux-1", 3*time.Hour)
	w.r.Ports.Macs = []ports.MacPorts{{Name: heroPro, ThisMac: true, Shared: []ports.SharedPort{{VM: "linux-1", GuestPort: 5173}}, Ports: []ports.Port{
		{Machine: machine.Ref{Name: "infra", On: heroPro}, Port: 20241, Process: "connector", Reach: ports.ThisMachine, Device: true},
	}}}
	until := heroNow.Add(4*time.Hour + 20*time.Minute)
	w.r.Awake = &awake.Report{Mac: heroPro, ThisMac: true, Engine: awake.EngineRunning, Session: &awake.Session{Until: until}}
	p := homeOf(t, w)
	if got := tile(t, p, "Ports"); got.Number.N != 1 || got.Word != "shared" || got.Nav != PortsID {
		t.Fatalf("ports %+v", got)
	}
	if got := tile(t, p, "Accounts"); got.Number.N != 1 || got.Word != "shared" {
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
