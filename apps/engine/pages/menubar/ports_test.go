package menubar

import (
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/ports"
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
// offers Declare public..., the hint that names that port's declare leaf,
// and Show process, a terminal on station ports of that Mac.
func TestAnExposedPortOffersDeclarePublicAndItsProcess(t *testing.T) {
	f := portFleet()
	f.Macs[1].Ports = append(f.Macs[1].Ports, ports.Port{Machine: coremachine.Ref{Name: miniMac}, Port: 10250, Process: "kubelet", Binds: []string{"*"}, Reach: ports.Outside})
	p := fleetPorts(t, f, PortsOf{})
	rows := rowsOf(p, SectionExposed)
	if len(rows) != 2 || rows[0].Label != "5000" || rows[0].Cells[0].Text != "AirPlay Receiver" || rows[0].Detail.Items[0].Text != page.ThisMac {
		t.Fatalf("exposed %+v", rows)
	}
	got := make([][]string, 0, len(rows))
	lines := make([]string, 0, len(rows))
	for _, r := range rows {
		declare, show := action(p, r.Menu[0]), action(p, r.Menu[1])
		if declare == nil || declare.Sheet != page.SheetHint+HintDeclarePublic || show == nil || show.Label != "Show process" {
			t.Fatalf("%s offers %q", r.ID, r.Menu)
		}
		lines = append(lines, declare.Form.Fields[0].Hint.Command)
		got = append(got, show.Run.Argv)
	}
	if want := [][]string{{"ports"}, {"ports", "--on", miniMac}}; !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("Show process runs %q, want %q", got, want)
	}
	if want := []string{"station ports declare 5000 --via <route>", "station ports declare " + miniMac + " 10250 --via <route>"}; !slices.Equal(lines, want) {
		t.Fatalf("Declare public names %q, want %q", lines, want)
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

// invariant: SPEC A4: Private · on your tailnet carries its info button,
// "About private links", on every page that lists it: all machines, a
// Mac's, a workspace's with nothing shared yet.
func TestThePrivateCardSaysWhatAPrivateLinkIs(t *testing.T) {
	f := portFleet()
	for name, p := range map[string]page.Page{
		"all machines": fleetPorts(t, f, PortsOf{}),
		miniMac:        macPageOf(t, miniMac, func(m *Machine) { m.Ports = &f }),
		"review-bot":   macPageOf(t, "review-bot", func(m *Machine) { m.Ports = &f }),
	} {
		about := sectionOf(p, SectionPrivate).About
		if about == nil || about.Label != "About private links" ||
			about.Text != "Private links open only on devices signed in to your tailnet; Public routes are a Cloudflare Tunnel or a Funnel you declared." {
			t.Errorf("%s's Private card explains %+v", name, about)
		}
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
// holds Share a port..., Stop and Delete.
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
	if c := rows[0].Cells; len(c) != 1 || c[0].Text != "private" {
		t.Fatalf("a share's card line names its audience: %+v", c)
	}
	menu := action(p, ActionMenu)
	if menu == nil || !slices.Equal(menuWords(p, menu.Submenu), []string{"linux-1 on This Mac", "New agent...", "Copy ssh command", "Share a port...", "Lend an account...", "-", "Stop...", "-", "Delete... (red)"}) {
		t.Fatalf("menu %q", menuWords(p, menu.Submenu))
	}
	f.Macs[0].Shared, f.Macs[0].Ports = nil, nil
	p = macPageOf(t, "linux-1", func(m *Machine) { m.All, m.Ports = fleet, &f })
	if items := sectionOf(p, SectionPorts).Items; len(items) != 1 || items[0].Empty == nil || items[0].Empty.Text != "Nothing shared yet" {
		t.Fatalf("an empty card %+v", items)
	}
}

// invariant: what a full VM refuses is one line with its count, and a list
// of thousands of listeners shows catalog.PortRowsShown rows and "N more".
func TestAFullVMAndAThousandListenersStayBounded(t *testing.T) {
	linux1 := coremachine.Ref{Name: "linux-1", On: proMac}
	heard := make([]ports.Port, 0, 2006)
	for i := range 2006 {
		heard = append(heard, ports.Port{Machine: linux1, Port: 3000 + i, Process: "node", Binds: []string{"0.0.0.0"}, Reach: ports.Tailnet})
	}
	f := ports.AllMachines{Macs: []ports.MacPorts{{Name: proMac, ThisMac: true, Reachable: true, HTTPS: ports.HTTPSOff, Ports: heard,
		NotShared: []ports.NotShared{{VM: "linux-1", Count: 1986, Detail: "linux-1 has 20 shared ports, the most a VM may"}}}}}
	p := fleetPorts(t, f, PortsOf{})
	listening := sectionOf(p, SectionListening)
	var rows int
	for _, it := range listening.Items {
		if it.Row != nil {
			rows++
		}
	}
	if rows != catalog.PortRowsShown {
		t.Fatalf("%d rows drawn, want %d", rows, catalog.PortRowsShown)
	}
	if last := listening.Items[len(listening.Items)-1]; last.Empty == nil || last.Empty.Text != catalog.MoreRows(2006-catalog.PortRowsShown) {
		t.Fatalf("last item %+v, want the line of the rest", last)
	}
	if listening.Caption[0].Text != "2006" {
		t.Fatalf("caption %v, want the whole count", listening.Caption)
	}
	if row := rowsOf(p, SectionPrivate)[0]; row.Label != "1986 ports not shared" || words(row.Detail) != "linux-1 has 20 shared ports, the most a VM may" {
		t.Fatalf("row %+v", row)
	}
}

// invariant: a capped section keeps the tally of every row it sums up.
func TestACappedCardKeepsTheTallyOfItsWholeCount(t *testing.T) {
	linux1 := coremachine.Ref{Name: "linux-1", On: proMac}
	heard := make([]ports.Port, 0, 120)
	for i := range 60 {
		heard = append(heard,
			ports.Port{Machine: linux1, Port: 3000 + i, Process: "node", Binds: []string{"0.0.0.0"}, Reach: ports.Outside},
			ports.Port{Machine: linux1, Port: 4000 + i, Process: "node", Binds: []string{"0.0.0.0"}, Reach: ports.Outside, Declared: &ports.Declaration{Machine: "linux-1", Port: 4000 + i, Via: catalog.RouteLAN}})
	}
	f := ports.AllMachines{Macs: []ports.MacPorts{{Name: proMac, ThisMac: true, Reachable: true, HTTPS: ports.HTTPSOff, Ports: heard}}}
	p := fleetPorts(t, f, PortsOf{})
	for _, id := range []string{SectionExposed, SectionPublic} {
		s := sectionOf(p, id)
		if s == nil || len(s.Counts) != 1 || s.Counts[0].N != 60 {
			t.Errorf("%s counts %+v, want the whole 60", id, s)
		}
	}
}

func TestAVMsQuietListenersAreOneLineOfItsCard(t *testing.T) {
	f := ports.AllMachines{Macs: []ports.MacPorts{{Name: proMac, ThisMac: true, Reachable: true, HTTPS: ports.HTTPSOff, Quiet: []ports.Quiet{{VM: "linux-1", Count: 3}}}}}
	p := fleetPorts(t, f, PortsOf{})
	rows := rowsOf(p, SectionPrivate)
	if len(rows) != 1 || rows[0].Label != "3 listeners on ports the system keeps for itself" || words(rows[0].Detail) != "station ports share linux-1 <port> to share one" {
		t.Fatalf("rows %+v", rows)
	}
}

// invariant: a share sits in the card of its audience: public in Public,
// local in This Mac only, the rest in Private; a listener outside the
// rights is flagged among the listeners, never drawn as shared.
func TestAShareSitsInTheCardOfItsAudience(t *testing.T) {
	f := portFleet()
	f.Macs[0].Shared = []ports.SharedPort{
		{VM: "linux-1", GuestPort: 3000, Reserved: 40001, State: ports.PhaseShared, To: "public", URLs: ports.URLSet{Public: "https://pro.ts.net:8443"}},
		{VM: "linux-1", GuestPort: 3001, Reserved: 40002, State: ports.PhaseShared, To: "local", URLs: ports.URLSet{Public: "http://127.0.0.1:40002"}},
		{VM: "linux-1", GuestPort: 3002, Reserved: 40003, State: ports.PhaseShared, To: "private", URLs: ports.URLSet{Public: "https://pro.ts.net:40003"}},
		{VM: "linux-1", GuestPort: 8443, State: ports.PhaseNotAllowed, Detail: "listens, not allowed (open: 3000-3999)"},
	}
	p := fleetPorts(t, f, PortsOf{})
	for section, want := range map[string][]string{
		SectionPublic: {"linux-1:3000", "infra:443"}, SectionLocal: {"linux-1:3001"}, SectionPrivate: {"linux-1:3002", "infra:6443"},
	} {
		if got := ids(rowsOf(p, section)); !slices.Equal(got, want) {
			t.Errorf("%s lists %q, want %q", section, got, want)
		}
	}
	flagged := rowsOf(p, SectionListening)[0]
	if flagged.ID != "linux-1:8443" || flagged.Mark != page.Attention || flagged.Detail.Items[1].Text != "open: 3000-3999" || flagged.Cells[0].Text != catalog.NotAllowedWord || action(p, flagged.Verb) == nil {
		t.Fatalf("flagged %+v", flagged)
	}
}

// invariant: a share draws what its state says, in every audience: served,
// it opens its link; waiting, refused, local only, not listening, stopped
// or issuing, it is flagged with its cause and offers Stop sharing alone;
// outside the rights it is a listener, flagged, offered Share.
func TestAShareOpensItsLinkOnlyWhileItIsServed(t *testing.T) {
	stoppedVM := catalog.StoppedDetail("linux-1")
	states := []struct {
		name   string
		share  ports.SharedPort
		opens  bool
		mark   page.Mark
		flag   string
		listen bool
	}{
		{"served", ports.SharedPort{Reserved: 40001, State: ports.PhaseShared}, true, page.OK, "", false},
		{"waiting", ports.SharedPort{State: ports.PhaseRefused, Detail: "no Funnel port is left"}, false, page.Attention, "waiting", false},
		{"unreserved", ports.SharedPort{State: ports.PhaseShared}, false, page.Attention, "waiting", false},
		{"local only", ports.SharedPort{Reserved: 40001, State: ports.PhaseShared, Detail: "serve: Funnel port 443 already serves an entry station did not make"}, false, page.Attention, "local only", false},
		{"not listening", ports.SharedPort{Reserved: 40001, State: ports.PhaseWaiting}, false, page.Absent, "not listening", false},
		{"stopped", ports.SharedPort{Reserved: 40001, State: ports.PhaseWaiting, Detail: stoppedVM}, false, page.Absent, stoppedVM, false},
		{"engine stopped", ports.SharedPort{Reserved: 40001, State: ports.PhaseUnknown}, false, page.Attention, "not shared while the engine is stopped", false},
		{"issuing", ports.SharedPort{Reserved: 40001, State: ports.PhaseShared, Detail: catalog.IssuingDetail}, false, page.Attention, catalog.IssuingDetail, false},
		{"not allowed", ports.SharedPort{State: ports.PhaseNotAllowed, Detail: "listens, not allowed (open: 3000-3999)"}, false, page.Attention, catalog.NotAllowedWord, true},
	}
	cards := map[catalog.Audience]string{"public": SectionPublic, "private": SectionPrivate, "local": SectionLocal}
	for _, st := range states {
		for audience, section := range cards {
			share := st.share
			share.VM, share.GuestPort, share.HTTP, share.To = "linux-1", 3000, true, audience
			share.URLs = ports.URLSet{Public: "https://pro.ts.net:443"}
			f := portFleet()
			f.Macs[0].Shared = []ports.SharedPort{share}
			p := fleetPorts(t, f, PortsOf{})
			card := section
			if st.listen {
				card = SectionListening
			}
			var row *page.Row
			for _, r := range rowsOf(p, card) {
				if r.ID == "linux-1:3000" {
					row = &r
				}
			}
			if row == nil {
				t.Errorf("%s, %s: no row in %s", st.name, audience, card)
				continue
			}
			verb := action(p, row.Verb)
			switch {
			case st.opens:
				if verb == nil || verb.Run == nil || verb.Run.Open == "" || row.Mark != page.OK {
					t.Errorf("%s, %s: served row %+v verb %+v", st.name, audience, row, verb)
				}
			case st.listen:
				if verb == nil || verb.Run == nil || verb.Run.Call != CallSharePort {
					t.Errorf("%s, %s: no Share on %+v", st.name, audience, row)
				}
			default:
				if row.Verb != "" || row.Quick != "" || len(row.Menu) != 1 || row.Mark != page.Attention && row.Mark != page.Absent {
					t.Errorf("%s, %s: unserved row draws a link: %+v", st.name, audience, row)
				}
				for _, a := range p.Actions {
					if a.Run != nil && (a.Run.Open != "" || a.Run.Copy != "") && slices.Contains(a.Rows, row.ID) {
						t.Errorf("%s, %s: %s on an unserved share", st.name, audience, a.ID)
					}
				}
			}
			if st.mark != row.Mark || st.flag != "" && (len(row.Cells) == 0 || row.Cells[0].Text != st.flag) {
				t.Errorf("%s, %s: mark %q cells %+v, want %q %q", st.name, audience, row.Mark, row.Cells, st.mark, st.flag)
			}
		}
	}
}

// invariant: a listener outside the rights is offered by the Share a port
// sheet with the listeners that wait, and a share past the cap does not
// shorten the count of This Mac only.
func TestANotAllowedListenerIsOfferedAndTheLocalCountKeepsAllShares(t *testing.T) {
	f := portFleet()
	f.Macs[0].Shared = []ports.SharedPort{{VM: "linux-1", GuestPort: 8443, State: ports.PhaseNotAllowed, Detail: "listens, not allowed (open: 3000-3999)"}}
	b := portsOf(f, portsScope{Self: proMac})
	if !slices.ContainsFunc(b.Listeners, func(it page.Item) bool { return it.Row.ID == "linux-1:8443" }) {
		t.Fatalf("the sheet's listeners %q", ids(rowsFrom(b.Listeners)))
	}
	f.Macs[0].Shared = nil
	for i := range catalog.PortRowsShown + 5 {
		f.Macs[0].Shared = append(f.Macs[0].Shared, ports.SharedPort{VM: "linux-1", GuestPort: 3000 + i, Reserved: 40000 + i, State: ports.PhaseShared, To: "local"})
	}
	b = portsOf(f, portsScope{Self: proMac})
	for _, s := range b.Sections {
		if s.ID == SectionLocal && (s.Caption[0].Text != strconv.Itoa(catalog.PortRowsShown+5) || !s.Uncounted) {
			t.Fatalf("local caption %+v uncounted %v", s.Caption, s.Uncounted)
		}
	}
}

func rowsFrom(items []page.Item) []page.Row {
	out := make([]page.Row, 0, len(items))
	for _, it := range items {
		out = append(out, *it.Row)
	}
	return out
}
