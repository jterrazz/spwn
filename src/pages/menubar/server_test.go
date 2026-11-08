package menubar

import (
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/machine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/ports"
)

var serverNow = time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)

func serverOn(edit func(*model.VM)) page.Page { return serverServing(nil, edit) }

func serverServing(served *ports.AllMachines, edit func(*model.VM)) page.Page {
	vm := model.VM{Name: "infra", Lifetime: model.AlwaysOn, State: model.Running, Want: "running", IP: "192.168.64.7", SSH: true, Guest: "connected",
		Check: "https://jterrazz.com/healthz", BootAt: serverNow.Add(-12 * 24 * time.Hour), Resources: model.Resources{CPUs: 8, MemoryBytes: 24 << 30},
		CheckStatus: model.CheckStatus{State: model.CheckPassing, At: serverNow.Add(-40 * time.Second), Answer: "200"}}
	for i := range 3 {
		vm.Latency = append(vm.Latency, model.LatencyPoint{At: serverNow.Add(time.Duration(i-2) * time.Minute), Took: time.Duration(30+i*10) * time.Millisecond})
	}
	if edit != nil {
		edit(&vm)
	}
	fleet := &model.AllMachines{Read: serverNow, Macs: []model.MacView{
		{Name: "jterrazz-macbook-pro", ThisMac: true, Reachable: true},
		{Name: "jterrazz-mac-mini", Reachable: true, VMs: []model.VM{vm}},
	}}
	page.Strict(true)
	defer page.Strict(false)
	return MachinePage(Machine{Name: "infra", All: fleet, Ports: served, Now: serverNow, Zone: time.UTC, Asks: func(string, machine.Kind) string { return LevelYesNo }})
}

func healthHead(t *testing.T, p page.Page) page.Group {
	t.Helper()
	s := sectionOf(p, SectionHealth)
	if s == nil || len(s.Items) != 1 || s.Items[0].Group == nil {
		t.Fatalf("no Health card: %+v", s)
	}
	return *s.Items[0].Group
}

// invariant: SPEC A4: a healthy server's Health is folded behind Healthy,
// 4 of 4, unfolding to its four checks, each with what it answered, then
// Restart and Stop.
func TestAHealthyServerFoldsItsFourChecksBehindItsCount(t *testing.T) {
	g := healthHead(t, serverOn(nil))
	if g.Head.Label != "Healthy" || words(g.Head.Cells[0]) != "4 of 4" || g.Fold != page.Folded {
		t.Fatalf("head %q %+v fold %q, want Healthy, 4 of 4, folded", g.Head.Label, g.Head.Cells, g.Fold)
	}
	said := make([]string, 0, len(g.Items))
	for _, it := range g.Items {
		said = append(said, it.Row.Label+"="+words(it.Row.Detail))
	}
	if want := []string{"Running=running", "Has an address=192.168.64.7", "SSH answers=", "Check passes=200 | 50 ms"}; !slices.Equal(said, want) {
		t.Fatalf("checks %q, want %q", said, want)
	}
	if !slices.Equal(g.Head.Menu, []string{ActionRestart, ActionStop}) {
		t.Fatalf("unfolded verbs %q, want Restart and Stop", g.Head.Menu)
	}
}

// invariant: SPEC B5.11, A9 3-9: a restart opens Health by itself and
// counts the step running: those before it passed, it checking, the rest
// next; the gauges wait and Open in Terminal waits for SSH.
func TestARestartCountsTheStepRunningAndTheRestWait(t *testing.T) {
	p := serverOn(func(vm *model.VM) { vm.Restart = &model.RestartWalk{Step: 2, At: serverNow.Add(-time.Second)} })
	g := healthHead(t, p)
	if g.Head.Label != "Restarting" || words(g.Head.Cells[0]) != "step 2 of 4" || g.Fold != page.Open || g.Head.Detail.Text != "started 1 s ago" {
		t.Fatalf("head %q %q fold %q", g.Head.Label, g.Head.Detail.Text, g.Fold)
	}
	signals := make([]page.Signal, 0, len(g.Items))
	for _, it := range g.Items {
		signals = append(signals, it.Row.Signal)
	}
	if want := []page.Signal{page.SignalWell, page.SignalActive, page.SignalQuiet, page.SignalQuiet}; !slices.Equal(signals, want) {
		t.Fatalf("steps %q, want passed, running, next, next", signals)
	}
	for _, it := range sectionOf(p, SectionGauges).Items {
		if it.Gauge.Value.Kind != page.KindPending {
			t.Fatalf("gauge %q reads %+v while it restarts, want it waiting", it.Gauge.Title, it.Gauge.Value)
		}
	}
	if p.Footer.Note != "Opens once SSH answers" || p.Footer.Copy != nil {
		t.Fatalf("footer %+v, want it to wait for SSH", p.Footer)
	}
}

// invariant: SPEC B5.5, A9 1-7: a down server says since when and why, with
// Restart run at once, and the last lines its console kept, each as its
// time, its source and what it said, the cause line counted.
func TestADownServerSaysWhyWithItsLastLines(t *testing.T) {
	p := serverOn(func(vm *model.VM) {
		vm.State, vm.BootAt, vm.Cause = model.Stopped, time.Time{}, "out of memory"
		vm.CheckStatus = model.CheckStatus{State: model.CheckFailing, Since: serverNow.Add(-4 * time.Minute)}
		vm.LastLines = []string{
			"[ 4126.903117] k3s[812]: etcd: slow fdatasync, took 1.9 s",
			"[ 4127.551204] Out of memory: Killed process 812 (k3s-server)",
			"[ 4127.611008] oom_reaper: reaped process 812 (k3s-server)",
		}
	})
	g := healthHead(t, p)
	restart := g.Head.Verb
	if g.Head.Label != "Down since 10:26" || g.Head.Detail.Text != "Out of memory" || g.Head.Mark != page.Broken {
		t.Fatalf("banner %q %q %q", g.Head.Label, g.Head.Detail.Text, g.Head.Mark)
	}
	for _, a := range p.Actions {
		if a.ID == restart && (a.Confirm != nil || a.Run == nil || a.Run.Call != CallRestartVM) {
			t.Fatalf("the banner's Restart %+v asks or runs no restart", a)
		}
	}
	log := sectionOf(p, SectionLog)
	if log == nil {
		t.Fatal("no Last lines card")
	}
	l := log.Items[0].Log
	want := []string{"10:25:59 k3s etcd: slow fdatasync, took 1.9 s", "10:25:59 kernel Out of memory: Killed process 812 (k3s-server)", "10:26:00 oom_reaper reaped process 812 (k3s-server)"}
	if !slices.Equal(l.Lines, want) || l.Cause != 2 || l.More != ActionLog {
		t.Fatalf("log %+v, want %q with the cause line second", l, want)
	}
	if sectionOf(p, SectionLatency) != nil {
		t.Fatal("a down server shows its service's latency")
	}
	if p.Footer.Note != "Needs infra running" {
		t.Fatalf("footer %+v", p.Footer)
	}
}

// invariant: SPEC A4: a server its owner stopped is no need: it says
// Stopped by you with Start, and nothing restarts it.
func TestAServerItsOwnerStoppedOffersStart(t *testing.T) {
	p := serverOn(func(vm *model.VM) { vm.State, vm.Want = model.Stopped, "stopped" })
	g := healthHead(t, p)
	if g.Head.Label != "Stopped" || g.Head.Verb != ActionStartVM || g.Head.Glyph.Dot != page.DotSquare {
		t.Fatalf("head %+v, want Stopped with Start and the square dot", g.Head)
	}
	if sectionOf(p, SectionLog) != nil {
		t.Fatal("a stopped server shows why it went down")
	}
}

// invariant: SPEC B5.4, D1: the latency card is the site its check asks, its
// latency now, its p50 and its peak with when; no request rate.
func TestTheLatencyCardSaysLatencyOnly(t *testing.T) {
	s := sectionOf(serverOn(nil), SectionLatency)
	if s == nil || s.Title != "jterrazz.com" || s.Caption[0].Text != "50 ms" {
		t.Fatalf("service %+v", s)
	}
	if got := s.Items[0].Fact.Key + ": " + words(s.Items[0].Fact.Value); got != "health check: p50 40 ms | peak 50 ms, 1 s ago" {
		t.Fatalf("service words %q", got)
	}
}

// invariant: SPEC A4: the server's facts name the Mac it is on as a link
// that pushes that Mac's page.
func TestAServersFactsLinkToTheMacItIsOn(t *testing.T) {
	for _, part := range sectionOf(serverOn(nil), SectionSummary).Caption {
		if part.Text == "on jterrazz-mac-mini" {
			if part.Nav != machinePage("jterrazz-mac-mini") {
				t.Fatalf("on jterrazz-mac-mini pushes %q, want the Mac's page", part.Nav)
			}
			return
		}
	}
	t.Fatal("the facts do not say the Mac it is on")
}

// invariant: SPEC A4: each port a healthy server serves copies, from its
// row, the address that reaches it: a public one the site its check asks
// through its route, one on its tailnet device its name and port.
func TestEveryPortAServerServesCopiesTheAddressThatReachesIt(t *testing.T) {
	infra := machine.Ref{Name: "infra", On: "jterrazz-mac-mini"}
	served := &ports.AllMachines{Macs: []ports.MacPorts{{Name: "jterrazz-mac-mini", Reachable: true, Ports: []ports.Port{
		{Machine: infra, Port: 443, Process: "nginx", Reach: ports.Outside, Declared: &ports.Declaration{Machine: "infra", Port: 443, Via: catalog.RouteCloudflareTunnel}},
		{Machine: infra, Port: 6443, Process: "kube-apiserver", Reach: ports.Tailnet, Device: true},
	}}}}
	p := serverServing(served, nil)
	copied := map[string]string{}
	for _, it := range sectionOf(p, SectionPorts).Items {
		a := action(p, it.Row.Quick)
		if a == nil || a.Run == nil || a.Run.Copy == "" {
			t.Fatalf("row %q copies nothing: %+v", it.Row.Label, a)
		}
		copied[it.Row.Label] = a.Run.Copy
	}
	if want := map[string]string{"Traefik": "https://jterrazz.com", "Kubernetes API": "infra:6443"}; !maps.Equal(copied, want) {
		t.Fatalf("copied %q, want %q", copied, want)
	}
}

// invariant: SPEC A4 (Down): a server that cannot answer keeps its last
// CPU and memory, said last, and when it was last seen.
func TestADownServersGaugesSayTheyAreItsLast(t *testing.T) {
	p := serverOn(func(vm *model.VM) { vm.State, vm.BootAt = model.Stopped, time.Time{} })
	gauges := sectionOf(p, SectionGauges).Items
	said := make([]string, 0, len(gauges))
	for _, it := range gauges {
		said = append(said, strings.TrimSpace(it.Gauge.Title+" "+it.Gauge.Note))
		if !it.Gauge.Stale {
			t.Fatalf("gauge %q is not stale", it.Gauge.Title)
		}
	}
	if want := []string{"CPU last", "Memory last", "Last seen"}; !slices.Equal(said, want) {
		t.Fatalf("gauges %q, want %q", said, want)
	}
	cpu := sectionOf(serverOn(nil), SectionGauges).Items[0].Gauge
	if cpu.Title != "CPU" || cpu.Note != "8 cores" {
		t.Fatalf("a healthy CPU gauge says %q %q, want its cores", cpu.Title, cpu.Note)
	}
}
