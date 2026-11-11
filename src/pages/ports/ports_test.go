package ports

import (
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/ports"
)

func mac(listeners ...ports.Port) ports.AllMachines {
	return ports.AllMachines{Macs: []ports.MacPorts{{Name: "mac", ThisMac: true, Reachable: true, Engine: ports.EngineInfo{Running: true}, Ports: listeners}}}
}

func many(n int, reach ports.Reach, bind string) []ports.Port {
	out := make([]ports.Port, 0, n)
	for i := range n {
		out = append(out, ports.Port{Machine: machine.Ref{Name: "mac"}, Port: 3000 + i, Process: "node", Binds: []string{bind}, Reach: reach})
	}
	return out
}

func sectionOf(p page.Page, id string) *page.Section {
	for i := range p.Sections {
		if p.Sections[i].ID == id {
			return &p.Sections[i]
		}
	}
	return nil
}

// invariant: the count of what listens on its machine only is the whole of
// it, however many rows a list may draw.
func TestTheCountOfLocalOnlyListenersIsTakenBeforeTheCap(t *testing.T) {
	p := List(ports.Listing{AllMachines: mac(many(100, ports.ThisMachine, "127.0.0.1")...)}, catalog.Catalog{}, page.Context{Machine: "mac"})
	ranges := sectionOf(p, "ranges")
	if ranges == nil || len(ranges.Items) != 1 || ranges.Items[0].Note.Text != "100 more listen on their machine only" {
		t.Fatalf("ranges %+v, want the whole 100", ranges)
	}
}

// invariant: a capped section keeps the tally of every row, not of the rows drawn.
func TestACappedSectionKeepsTheTallyOfItsWholeCount(t *testing.T) {
	p := Listed(ports.Listing{AllMachines: mac(many(100, ports.ThisMachine, "127.0.0.1")...)}, catalog.Catalog{}, page.Context{Machine: "mac"}, true)
	local := sectionOf(p, "local-only")
	if local == nil || len(local.Counts) != 1 || local.Counts[0].N != 100 {
		t.Fatalf("local-only counts %+v, want 100", local)
	}
	var rows int
	for _, it := range local.Items {
		if it.Row != nil {
			rows++
		}
	}
	if rows != catalog.PortRowsShown {
		t.Fatalf("%d rows drawn, want %d", rows, catalog.PortRowsShown)
	}
}

func TestAVMsQuietListenersAreOneLine(t *testing.T) {
	f := mac()
	f.Macs[0].Quiet = []ports.Quiet{{VM: "linux-1", Count: 3}}
	p := List(ports.Listing{AllMachines: f}, catalog.Catalog{}, page.Context{Machine: "mac"})
	private := sectionOf(p, "private")
	if private == nil || len(private.Items) != 1 || private.Items[0].Row == nil {
		t.Fatalf("private %+v, want one line", private)
	}
	row := private.Items[0].Row
	if row.ID != "linux-1:quiet" || row.Mark != page.Attention || row.Cells[3].Items[0].Text != "3 listeners on ports the system keeps for itself" || row.Cells[3].Items[1].Text != "station ports share linux-1 <port> to share one" {
		t.Fatalf("row %+v", row)
	}
}

// invariant: a listener outside its VM's rights reads not-allowed in
// station.ports.list/v1, though it is never reserved.
func TestAListenerOutsideItsRightsReadsNotAllowed(t *testing.T) {
	got := PortObject(ports.SharedPort{VM: "ws1", GuestPort: 8443, State: ports.PhaseNotAllowed, Detail: "outside its rights"}).State
	if got != "not-allowed" {
		t.Fatalf("state %q, want not-allowed", got)
	}
}

// invariant: a shared port's class in station.ports.list/v1 is its
// audience's: public for one open to the internet, private for the others,
// and `to` names the audience itself.
func TestASharedPortsClassFollowsItsAudience(t *testing.T) {
	for to, class := range map[string]string{"public": "public", "private": "private", "local": "private"} {
		got := PortObject(ports.SharedPort{VM: "ws1", GuestPort: 3000, Reserved: 40000, To: to})
		if got.Class != class || got.To != to {
			t.Fatalf("to %q: class %q, to %q, want class %q", to, got.Class, got.To, class)
		}
	}
}

func TestASharedPortsRowSaysWhatProbablyServesIt(t *testing.T) {
	row := Row(ports.SharedPort{VM: "ws1", GuestPort: 5173, Reserved: 40000, Label: "Vite"})
	if row.Detail.Text != "probably Vite" {
		t.Fatalf("detail %+v, want probably Vite", row.Detail)
	}
}

// invariant: a share prints its URL only while a reservation holds it: not
// a refused one, not a listener outside the rights, not one the engine
// stopped.
func TestAShareWithNoReservationPrintsNoURL(t *testing.T) {
	url := ports.URLSet{Public: "https://mac.ts.net:443"}
	for state, want := range map[ports.Phase]bool{ports.PhaseShared: true, ports.PhaseWaiting: true, ports.PhaseRefused: false, ports.PhaseNotAllowed: false, ports.PhaseUnknown: false} {
		row := Row(ports.SharedPort{VM: "vm", GuestPort: 3000, Reserved: 40000, State: state, URLs: url})
		if got := row.Cells[2].Kind == page.KindURL; got != want {
			t.Errorf("%s: url shown %v, want %v", state, got, want)
		}
	}
}
