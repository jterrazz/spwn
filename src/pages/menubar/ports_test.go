package menubar

import (
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/ports"
)

func portFleet() ports.AllMachines {
	linux1, infra, linux3 := coremachine.Ref{Name: "linux-1", On: proMac}, coremachine.Ref{Name: "infra", On: miniMac}, coremachine.Ref{Name: "linux-3", On: miniMac}
	listen := func(m coremachine.Ref, port int, process string) ports.Port {
		return ports.Port{Machine: m, Port: port, Process: process, Binds: []string{"0.0.0.0"}, Reach: ports.Tailnet}
	}
	return ports.AllMachines{Macs: []ports.MacPorts{
		{Name: proMac, ThisMac: true, Reachable: true, HTTPS: ports.HTTPSOff, Shared: []ports.SharedPort{
			{VM: "linux-1", GuestPort: 5173, Reserved: 40001, State: ports.PhaseShared, HTTP: true, URLs: ports.URLSet{Public: "http://pro.ts.net:40001"}, ReservedAt: macNow.Add(-time.Hour)},
		}, Ports: []ports.Port{listen(linux1, 4321, "astro"), {Machine: coremachine.Ref{Name: proMac}, Port: 5000, Process: "ControlCenter", Binds: []string{"*"}, Reach: ports.Outside}}},
		{Name: miniMac, Reachable: true, HTTPS: ports.HTTPSPort, Ports: []ports.Port{
			{Machine: infra, Port: 443, Process: "nginx", Binds: []string{"0.0.0.0"}, Reach: ports.Outside, Declared: &ports.Declaration{Machine: "infra", Port: 443, Via: catalog.RouteCloudflareTunnel, URL: "https://jterrazz.com"}},
			{Machine: infra, Port: 6443, Process: "kube-apiserver", Binds: []string{"0.0.0.0"}, Reach: ports.Tailnet, Device: true},
			listen(linux3, 5432, "postgres"),
		}},
	}}
}

func fleetPorts(t *testing.T, f ports.AllMachines, of PortsOf) page.Page {
	t.Helper()
	page.Strict(true)
	defer page.Strict(false)
	return Ports(ports.Listing{AllMachines: f}, of, page.Context{Machine: proMac})
}

func rowsOf(p page.Page, section string) []page.Row {
	s := sectionOf(p, section)
	if s == nil {
		return nil
	}
	var out []page.Row
	var walk func([]page.Item)
	walk = func(items []page.Item) {
		for _, it := range items {
			if it.Row != nil {
				out = append(out, *it.Row)
			}
			if it.Group != nil {
				walk(it.Group.Items)
			}
		}
	}
	walk(s.Items)
	return out
}

func ids(rows []page.Row) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.ID)
	}
	return out
}

// invariant: the Ports of all machines say what is exposed first, then what
// is public, private and only listening, each port in its one place: a
// share or what listens on a VM with its own tailnet device is private,
// what listens on a VM behind its Mac waits for Share.
func TestEveryPortOfTheFleetIsListedOnceInItsReach(t *testing.T) {
	p := fleetPorts(t, portFleet(), PortsOf{})
	var got []string
	for _, s := range p.Sections {
		got = append(got, s.ID)
	}
	if want := []string{SectionSummary, SectionExposed, SectionPublic, SectionPrivate, SectionListening}; !slices.Equal(got, want) {
		t.Fatalf("sections %q, want %q", got, want)
	}
	for section, want := range map[string][]string{
		SectionExposed: {proMac + ":5000"}, SectionPublic: {"infra:443"},
		SectionPrivate: {"linux-1:5173", "infra:6443"}, SectionListening: {"linux-1:4321", "linux-3:5432"},
	} {
		if got := ids(rowsOf(p, section)); !slices.Equal(got, want) {
			t.Errorf("%s lists %q, want %q", section, got, want)
		}
	}
	if facts := sectionOf(p, SectionSummary).Caption; len(facts) != 4 || facts[1].Text != "1 public" || facts[2].Text != "2 on your tailnet" ||
		facts[3].Text != "1 exposed" || facts[3].Signal != page.SignalFailing {
		t.Fatalf("facts %v", facts)
	}
}

// invariant: a web share opens in a click and offers its QR; a raw TCP
// port is copied; a listener is shared on the Mac its VM runs on.
func TestAPortRowIsItsOwnVerb(t *testing.T) {
	p := fleetPorts(t, portFleet(), PortsOf{})
	vite, kube := rowsOf(p, SectionPrivate)[0], rowsOf(p, SectionPrivate)[1]
	if open := action(p, vite.Verb); open == nil || open.Run.Open != "http://pro.ts.net:40001" || vite.Quick != ActionQR+"linux-1:5173" {
		t.Fatalf("Vite's row runs %+v with %q beside", open, vite.Quick)
	}
	if copyIt := action(p, kube.Verb); copyIt == nil || copyIt.Run.Copy != "infra:6443" {
		t.Fatalf("Kubernetes API's row runs %+v", copyIt)
	}
	postgres := rowsOf(p, SectionListening)[1]
	share := action(p, postgres.Verb)
	if share == nil || share.Label != VerbShare || share.Run.Call != CallSharePort || share.Run.In["VM"] != "linux-3" || share.Run.In["Port"] != "5432" ||
		share.Run.Target == nil || share.Run.Target.Name != miniMac {
		t.Fatalf("Postgres's Share runs %+v", share)
	}
	if postgres.Label != "Postgres" || vite.Label != "Vite" || kube.Label != "Kubernetes API" {
		t.Fatalf("the rows are named %q, %q, %q", postgres.Label, vite.Label, kube.Label)
	}
}

// invariant: SPEC A4: a public port names where the outside reaches it
// and by what route; its row opens that link, copies it, and offers its QR.
func TestAPublicPortOpensItsLink(t *testing.T) {
	p := fleetPorts(t, portFleet(), PortsOf{})
	traefik := rowsOf(p, SectionPublic)[0]
	if d := traefik.Detail.Items; len(d) != 2 || d[0].Text != "jterrazz.com" || d[1].Text != "Cloudflare Tunnel" {
		t.Fatalf("the public row says %+v", traefik.Detail)
	}
	if open := action(p, traefik.Verb); open == nil || open.Label != "Open Traefik" || open.Run.Open != "https://jterrazz.com" {
		t.Fatalf("the public row runs %+v", open)
	}
	if qr := action(p, traefik.Quick); qr == nil || qr.Label != "Show QR for Traefik" {
		t.Fatalf("the public row offers %+v beside", qr)
	}
	if copyIt := action(p, ActionCopyPort+"infra:443"); copyIt == nil || copyIt.Run.Copy != "https://jterrazz.com" {
		t.Fatalf("the public link copies %+v", copyIt)
	}
}

// invariant: a share also declared public is listed once, as public.
func TestAShareDeclaredPublicIsListedOnce(t *testing.T) {
	f := portFleet()
	f.Macs[1].Shared = []ports.SharedPort{{VM: "infra", GuestPort: 443, Reserved: 40003, State: ports.PhaseShared, HTTP: true, URLs: ports.URLSet{Public: "http://mini.ts.net:40003"}}}
	p := fleetPorts(t, f, PortsOf{})
	if got := ids(rowsOf(p, SectionPrivate)); slices.Contains(got, "infra:443") {
		t.Fatalf("Private lists %q, the public infra:443 among them", got)
	}
	if got := ids(rowsOf(p, SectionPublic)); !slices.Equal(got, []string{"infra:443"}) {
		t.Fatalf("Public lists %q", got)
	}
}

// invariant: SPEC B5.17: a port follows its VM: while infra is down its
// ports are greyed and said unreachable, with no verb to run.
func TestAPortOfADownServerIsUnreachableAndRunsNothing(t *testing.T) {
	f := portFleet()
	for i := range f.Macs[1].Ports {
		if f.Macs[1].Ports[i].Machine.Name == "infra" {
			f.Macs[1].Ports[i].Answer = ports.Unreachable
		}
	}
	p := fleetPorts(t, f, PortsOf{})
	for _, r := range []page.Row{rowsOf(p, SectionPublic)[0], rowsOf(p, SectionPrivate)[1]} {
		if r.Signal != page.SignalFaint || len(r.Cells) != 1 || r.Cells[0].Text != "unreachable" || r.Verb != "" || r.Quick != "" || r.Mac != "" {
			t.Errorf("%s reads %+v", r.ID, r)
		}
	}
}

// invariant: the exposed card names the app that listens and where, and
// offers Declare public… and Show process, a terminal on station ports
// of that Mac.
func TestAnExposedPortOffersDeclarePublicAndItsProcess(t *testing.T) {
	f := portFleet()
	f.Macs[1].Ports = append(f.Macs[1].Ports, ports.Port{Machine: coremachine.Ref{Name: miniMac}, Port: 10250, Process: "kubelet", Binds: []string{"*"}, Reach: ports.Outside})
	p := fleetPorts(t, f, PortsOf{})
	rows := rowsOf(p, SectionExposed)
	if len(rows) != 2 || rows[0].Label != "5000" || rows[0].Cells[0].Text != "AirPlay Receiver" || rows[0].Detail.Items[0].Text != page.ThisMac {
		t.Fatalf("exposed %+v", rows)
	}
	got := make([][]string, 0, len(rows))
	for _, r := range rows {
		show := action(p, r.Menu[1])
		if r.Menu[0] != ActionDeclarePublic || show == nil || show.Label != "Show process" {
			t.Fatalf("%s offers %q", r.ID, r.Menu)
		}
		got = append(got, show.Run.Argv)
	}
	if want := [][]string{{"ports"}, {"ports", "--on", miniMac}}; !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("Show process runs %q, want %q", got, want)
	}
}

// invariant: a Mac's Ports segment lists its ports and its VMs' alone,
// then its HTTPS links switch, which turns them the other way with
// ports.set-https on that Mac.
func TestAMacsPortsEndOnItsHTTPSLinksSwitch(t *testing.T) {
	f := portFleet()
	p := macPageOf(t, miniMac, func(m *Machine) { m.Ports = &f })
	if got := ids(rowsOf(p, SectionPrivate)); !slices.Equal(got, []string{"infra:6443"}) {
		t.Fatalf("mac-mini's private ports %q", got)
	}
	if sectionOf(p, SectionExposed) != nil || sectionOf(p, SectionPublic).Segment != SegmentPorts {
		t.Fatalf("mac-mini's ports %+v", p.Sections)
	}
	sw := rowsOf(p, SectionHTTPS)[0]
	turn := action(p, sw.Verb)
	if sw.Label != "HTTPS links on mac-mini" || sw.Cells[0].Kind != page.KindOn || turn.Run.Call != CallSetHTTPS || turn.Run.In["On"] != "false" ||
		turn.Run.Target == nil || turn.Run.Target.Name != miniMac {
		t.Fatalf("the switch reads %+v and runs %+v", sw, turn.Run)
	}
	p = macPageOf(t, proMac, func(m *Machine) { m.Ports = &f })
	sw = rowsOf(p, SectionHTTPS)[0]
	if turn := action(p, sw.Verb); sw.Cells[0].Kind == page.KindOn || turn.Run.In["On"] != "true" || turn.Run.Target != nil {
		t.Fatalf("This Mac's switch reads %+v and runs %+v", sw, turn.Run)
	}
}

// invariant: a workspace's Ports hides Public when it has none, and lists
// its listeners in its Private card, under their own words once it shares
// something; one with nothing at all says "Nothing shared yet".
func TestAWorkspacesPortsKeepItsListenersInItsCard(t *testing.T) {
	f := portFleet()
	p := macPageOf(t, "linux-3", func(m *Machine) { m.Ports = &f })
	if sectionOf(p, SectionPublic) != nil || sectionOf(p, SectionListening) != nil {
		t.Fatalf("linux-3's ports %+v", p.Sections)
	}
	if got := ids(rowsOf(p, SectionPrivate)); !slices.Equal(got, []string{"linux-3:5432"}) {
		t.Fatalf("linux-3's card %q", got)
	}
	f.Macs[1].Shared = []ports.SharedPort{{VM: "linux-3", GuestPort: 8080, Reserved: 40001, State: ports.PhaseShared}}
	p = macPageOf(t, "linux-3", func(m *Machine) { m.Ports = &f })
	items := sectionOf(p, SectionPrivate).Items
	if len(items) != 2 || items[1].Group == nil || items[1].Group.Head.Label != TitleListening {
		t.Fatalf("linux-3's card %+v", items)
	}
	p = macPageOf(t, "review-bot", func(m *Machine) { m.Ports = &f })
	if empty := sectionOf(p, SectionPrivate).Items[0].Empty; empty == nil || empty.Text != "Nothing shared yet" {
		t.Fatalf("review-bot's card %+v", sectionOf(p, SectionPrivate).Items)
	}
}

// invariant: a port shared a moment ago lands fresh on the page.
func TestAPortJustSharedIsFresh(t *testing.T) {
	f := portFleet()
	f.Macs[0].Shared = append(f.Macs[0].Shared, ports.SharedPort{VM: "linux-1", GuestPort: 4321, Reserved: 40002, State: ports.PhaseShared, HTTP: true,
		URLs: ports.URLSet{Public: "http://pro.ts.net:40002"}, ReservedAt: macNow.Add(-2 * time.Second)})
	fleet := macFleet()
	fleet.Macs[0].VMs = append(fleet.Macs[0].VMs, macFleet().Macs[1].VMs[0])
	fleet.Macs[0].VMs[0].Name = "linux-1"
	p := macPageOf(t, "linux-1", func(m *Machine) { m.All, m.Ports = fleet, &f })
	var fresh []string
	for _, r := range rowsOf(p, SectionPrivate) {
		if r.Fresh {
			fresh = append(fresh, r.ID)
		}
	}
	if !slices.Equal(fresh, []string{"linux-1:4321"}) {
		t.Fatalf("fresh %q", fresh)
	}
	fresh = nil
	for _, r := range rowsOf(fleetPorts(t, f, PortsOf{Now: macNow}), SectionPrivate) {
		if r.Fresh {
			fresh = append(fresh, r.ID)
		}
	}
	if !slices.Equal(fresh, []string{"linux-1:4321"}) {
		t.Fatalf("Ports of all machines: fresh %q", fresh)
	}
}

// invariant: a share's menu holds every verb of it: Open, Copy and its QR
// for a web share, Copy for a raw port, then Stop sharing, so a share is
// stopped from the panel wherever it is listed.
func TestAShareMenuStopsIt(t *testing.T) {
	f := portFleet()
	f.Macs[0].Shared = append(f.Macs[0].Shared, ports.SharedPort{VM: "linux-1", GuestPort: 7000, Reserved: 40002, State: ports.PhaseShared,
		URLs: ports.URLSet{Public: "pro.ts.net:40002"}})
	p := fleetPorts(t, f, PortsOf{})
	menus := map[string][]string{}
	for _, r := range rowsOf(p, SectionPrivate) {
		if r.Menu != nil {
			menus[r.ID] = r.Menu
		}
	}
	want := map[string][]string{
		"linux-1:5173": {ActionOpenPort + "linux-1:5173", ActionCopyPort + "linux-1:5173", ActionQR + "linux-1:5173", "-", ActionStopSharing + "linux-1:5173"},
		"linux-1:7000": {ActionCopyPort + "linux-1:7000", "-", ActionStopSharing + "linux-1:7000"},
	}
	if len(menus) != len(want) {
		t.Fatalf("menus %q", menus)
	}
	for id, menu := range want {
		if !slices.Equal(menus[id], menu) {
			t.Fatalf("%s's menu %q, want %q", id, menus[id], menu)
		}
	}
	stop := action(p, ActionStopSharing+"linux-1:7000")
	if stop == nil || stop.Run == nil || stop.Run.Call != CallStopSharing || stop.Run.In["Port"] != `"7000"` || stop.Confirm == nil {
		t.Fatalf("stop %+v", stop)
	}
}

func action(p page.Page, id string) *page.Action {
	for i := range p.Actions {
		if p.Actions[i].ID == id {
			return &p.Actions[i]
		}
	}
	return nil
}

// invariant: SPEC A4: a running workspace's Overview has its Ports card,
// its shares then its first listener offered with Share, and its menu
// holds Share a port…, Stop and Delete.
func TestAWorkspaceOverviewHasItsPortsCard(t *testing.T) {
	f := portFleet()
	f.Macs[0].Ports = append(f.Macs[0].Ports, ports.Port{Machine: coremachine.Ref{Name: "linux-1", On: proMac}, Port: 9229, Process: "node", Binds: []string{"0.0.0.0"}, Reach: ports.Tailnet})
	fleet := macFleet()
	fleet.Macs[0].VMs = append(fleet.Macs[0].VMs, macFleet().Macs[1].VMs[0])
	fleet.Macs[0].VMs[0].Name = "linux-1"
	p := macPageOf(t, "linux-1", func(m *Machine) { m.All, m.Ports = fleet, &f })
	card := sectionOf(p, SectionPorts)
	if card == nil || card.Segment != SegmentOverview || card.Title != TitlePorts {
		t.Fatalf("card %+v", card)
	}
	rows := rowsOf(p, SectionPorts)
	if got := ids(rows); !slices.Equal(got, []string{"linux-1:5173", "linux-1:4321"}) {
		t.Fatalf("card rows %q", got)
	}
	if rows[0].Verb != ActionOpenPort+"linux-1:5173" || rows[1].Verb != ActionSharePortOf+"linux-1:4321" {
		t.Fatalf("verbs %q %q", rows[0].Verb, rows[1].Verb)
	}
	menu := action(p, ActionMenu)
	if menu == nil || !slices.Equal(menu.Submenu, []string{ActionShare, ActionStop, ActionDelete, ActionOpenStation}) {
		t.Fatalf("menu %+v", menu)
	}
	f.Macs[0].Shared, f.Macs[0].Ports = nil, nil
	p = macPageOf(t, "linux-1", func(m *Machine) { m.All, m.Ports = fleet, &f })
	if items := sectionOf(p, SectionPorts).Items; len(items) != 1 || items[0].Empty == nil || items[0].Empty.Text != "Nothing shared yet" {
		t.Fatalf("an empty card %+v", items)
	}
}
