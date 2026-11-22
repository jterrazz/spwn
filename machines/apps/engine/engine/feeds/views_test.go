package feeds

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/engine/store"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

// value: allSnapshot with This Mac's ports, one a port of its server the
// tailnet knows as a device, its node, its sign-ins and its Awake, all read
// a minute before the snapshot's instant.
func ownSnapshot(now time.Time) *store.Snapshot {
	s := allSnapshot()
	s.At = now
	read := at.Add(-time.Minute)
	put := func(k store.Key, v any) { s.Slices[k] = slice(k, v, store.Fresh, read) }
	put(store.Of(store.FamilyPorts, thisMac), ports.MacPorts{Name: thisMac, ThisMac: true, Reachable: true, Shared: []ports.SharedPort{},
		Ports: []ports.Port{{Machine: machine.Ref{Name: "linux-1", On: thisMac}, Port: 5173}}})
	put(store.Of(store.FamilyNode, ""), ports.Status{Name: thisMac, Peers: []ports.Node{{Name: "linux-1"}}})
	put(store.Of(store.FamilySignins, thisMac), accounts.ProgramsRead{})
	put(store.Of(store.FamilySignins, "linux-1"), accounts.ProgramsRead{})
	put(store.Of(store.FamilyAwake, thisMac), awake.Report{Engine: awake.EngineRunning})
	return s
}

func projected(t *testing.T, s *store.Snapshot) map[string][]byte {
	t.Helper()
	pages := Pages{ThisMachine: page.Context{Machine: thisMac, Kind: "mac"}, Zone: time.UTC}
	out := map[string][]byte{}
	for name, v := range pages.Project(s, func(string) bool { return true }) {
		raw, err := json.Marshal(v.Value)
		if err != nil {
			t.Fatal(err)
		}
		out[name] = raw
	}
	return out
}

func TestTheOwnViewsAreTheSameBytesWhenOnlyTheClockMoves(t *testing.T) {
	first, later := projected(t, ownSnapshot(at)), projected(t, ownSnapshot(at.Add(10*time.Minute)))
	for _, name := range []string{ViewPorts, viewPortsOf + thisMac, viewPortsOf + "linux-1", ViewAccounts, viewAccountsOf + "linux-1", ViewAwake} {
		if first[name] == nil {
			t.Fatalf("no view %s in %v", name, first)
		}
		if !bytes.Equal(first[name], later[name]) {
			t.Errorf("%s moved with the clock alone:\n%s\n%s", name, first[name], later[name])
		}
	}
}

func TestThePortsViewMarksADeviceWithoutMovingTheSnapshot(t *testing.T) {
	s := ownSnapshot(at)
	pages := Pages{ThisMachine: page.Context{Machine: thisMac, Kind: "mac"}, Zone: time.UTC}
	listing, ok := pages.Project(s, func(name string) bool { return name == ViewPorts })[ViewPorts].Value.(ports.Listing)
	if !ok || !listing.AllMachines.Macs[0].Ports[0].Device {
		t.Fatalf("the server's port is not its own device: %+v", listing)
	}
	held, _ := s.Slices[store.Of(store.FamilyPorts, thisMac)].Value.(ports.MacPorts)
	if held.Ports[0].Device {
		t.Error("projecting the ports view marked the snapshot's own value")
	}
}
