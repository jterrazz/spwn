package feeds

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/engine/store"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

const (
	thisMac = "macbook-pro"
	mini    = "mac-mini"
	studio  = "mac-studio"
)

var at = time.Date(2026, 11, 3, 10, 30, 0, 0, time.UTC)

func slice(k store.Key, v any, state store.State, read time.Time) store.Slice {
	return store.Slice{Key: k, Value: v, State: state, ReadAt: read}
}

// value: a snapshot where This Mac answers with one VM, the mini went
// silent 5 min ago holding a server, and the studio was never heard.
func allSnapshot() *store.Snapshot {
	server := machines.VM{Name: "linux-1", Kind: catalog.Linux, Lifetime: machines.AlwaysOn, State: machines.Running}
	minis := store.Of(store.FamilyMac, mini)
	records := store.Of(store.FamilyRecords, "")
	own := store.Of(store.FamilyMac, thisMac)
	gone := at.Add(-5 * time.Minute)
	return &store.Snapshot{At: at, Slices: map[store.Key]store.Slice{
		own:   slice(own, machines.MacView{Name: thisMac, ThisMac: true, Reachable: true}, store.Fresh, at),
		minis: slice(minis, machines.MacView{Name: mini, Reachable: true, VMs: []machines.VM{server}}, store.Stale, gone),
		records: slice(records, map[string]machines.MacRecord{
			mini:   {Answered: gone.Add(-time.Minute)},
			studio: {VMs: []machines.SeenVM{{Name: "dev-1", Kind: catalog.Linux, Lifetime: machines.Persistent}}},
		}, store.Fresh, at),
	}}
}

func TestAMacGoneSilentKeepsTheVMsItLastSaidUnreachableAndOneNeverHeardIsDrawnFromItsRecord(t *testing.T) {
	all := view{s: allSnapshot(), this: thisMac, now: at}.machines()
	if len(all.Macs) != 3 || all.Macs[0].Name != thisMac {
		t.Fatalf("All machines holds %v, want This Mac then its two others", all.Macs)
	}
	for _, h := range all.Macs[1:] {
		switch h.Name {
		case mini:
			if h.Reachable || len(h.VMs) != 1 || h.VMs[0].Name != "linux-1" || h.VMs[0].Mac != mini {
				t.Errorf("the silent mini is %+v, want it unreachable with the server it last said", h)
			}
			if !h.Answered.Equal(at.Add(-5 * time.Minute)) {
				t.Errorf("the silent mini last answered %v, want its last reading", h.Answered)
			}
		case studio:
			if h.Reachable || len(h.VMs) != 1 || h.VMs[0].State != machines.UnknownState {
				t.Errorf("the studio never heard is %+v, want its remembered VM in an unknown state", h)
			}
		default:
			t.Errorf("All machines holds %q", h.Name)
		}
	}
}

func TestAMachinesPageOpensWholeFromTheStore(t *testing.T) {
	s := allSnapshot()
	ov := store.Of(store.FamilyOverview, thisMac)
	s.Slices[ov] = slice(ov, overview.Kept{Boot: at.Add(-time.Hour)}, store.Fresh, at)
	p := Pages{ThisMachine: page.Context{Machine: thisMac, Kind: "mac"}, Zone: time.UTC}.Machine(s, thisMac)
	if p.Header == nil {
		t.Fatalf("This Mac's page opened with no header: %+v", p)
	}
}

func TestAMachineGoneSilentKeepsTheActivityItLastSaid(t *testing.T) {
	s := allSnapshot()
	act := store.Of(store.FamilyActivity, "linux-1")
	s.Slices[act] = slice(act, activity.Report{}, store.Stale, at.Add(-5*time.Minute))
	v := view{s: s, this: thisMac, now: at}
	if got := v.activity("linux-1"); got == nil {
		t.Fatal("the silent server's activity was dropped, want the last it said")
	}
	if p := (Pages{ThisMachine: page.Context{Machine: thisMac, Kind: "mac"}, Zone: time.UTC}).Machine(s, "linux-1"); p.Header == nil {
		t.Fatalf("the silent server's page opened with no header: %+v", p)
	}
}

func TestOneSnapshotProjectsOnePageWhateverTheInstantItIsDrawnAt(t *testing.T) {
	s := allSnapshot()
	pages := Pages{ThisMachine: page.Context{Machine: thisMac, Kind: "mac"}, Zone: time.UTC}
	for _, draw := range []func(*store.Snapshot, string) page.Page{pages.Home, pages.Machine} {
		first, _ := json.Marshal(draw(s, "linux-1"))
		second, _ := json.Marshal(draw(s, "linux-1"))
		if !bytes.Equal(first, second) {
			t.Fatalf("one snapshot drew two pages:\n%s\n%s", first, second)
		}
	}
}
