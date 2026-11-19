package machines_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

func TestAVMsHoldSaysWhoKeepsItAndWhetherItIsSafeToDelete(t *testing.T) {
	for name, c := range map[string]struct {
		vm   machines.VM
		want machines.Hold
	}{
		"a server": {machines.VM{Lifetime: machines.AlwaysOn, State: machines.Running, Creator: "studio-os-fe"},
			machines.Hold{State: machines.HoldAlwaysOn, Sentence: "Always on. Stopping it needs its name typed."}},
		"a server being created": {machines.VM{Lifetime: machines.AlwaysOn, State: machines.Creating, Creator: "studio-os-fe"},
			machines.Hold{State: machines.HoldAlwaysOn, Sentence: "Always on. Stopping it needs its name typed."}},
		"its Mac not seen": {machines.VM{Lifetime: machines.AlwaysOn, State: machines.UnknownState},
			machines.Hold{State: machines.HoldNotSeen, Sentence: "Not seen: its Mac does not answer, so nothing says what holds it."}},
		"being created by a session": {machines.VM{Lifetime: machines.Persistent, State: machines.Creating, Creator: "studio-os-fe"},
			machines.Hold{State: machines.HoldHeld, By: "studio-os-fe", Sentence: "Being created by studio-os-fe. It cannot be deleted until its create ends."}},
		"being created by you": {machines.VM{Lifetime: machines.Ephemeral, State: machines.Creating, Creator: machines.ByOwner},
			machines.Hold{State: machines.HoldHeld, Sentence: "Being created. It cannot be deleted until its create ends."}},
		"an ephemeral one a session runs": {machines.VM{Lifetime: machines.Ephemeral, State: machines.Running, Creator: "studio-os-fe"},
			machines.Hold{State: machines.HoldHeld, By: "studio-os-fe", Sentence: "It is held by its lot, by studio-os-fe: work may be in progress."}},
		"a persistent one a session runs": {machines.VM{Lifetime: machines.Persistent, State: machines.Running, Creator: "studio-os-fe"},
			machines.Hold{State: machines.HoldYours, Sentence: "Yours. Stays until you delete it."}},
		"an ephemeral one stopped": {machines.VM{Lifetime: machines.Ephemeral, State: machines.Stopped, Creator: "studio-os-fe"},
			machines.Hold{State: machines.HoldFree, Sentence: "Stopped. Safe to delete."}},
		"an ephemeral one of yours": {machines.VM{Lifetime: machines.Ephemeral, State: machines.Running, Creator: machines.ByOwner},
			machines.Hold{State: machines.HoldYours, Sentence: "Yours. Deleted when it powers off."}},
		"a persistent one": {machines.VM{Lifetime: machines.Persistent, State: machines.Stopped, Creator: "studio-os-fe"},
			machines.Hold{State: machines.HoldYours, Sentence: "Yours. Stays until you delete it."}},
	} {
		if got := machines.HoldOf(c.vm); got != c.want {
			t.Errorf("%s: %+v; want %+v", name, got, c.want)
		}
	}
}

func TestAVMsSectionsAreItsKindsAndEachSectionHasAFlag(t *testing.T) {
	fields := map[string]bool{}
	typ := reflect.TypeFor[machines.Sections]()
	for i := range typ.NumField() {
		fields[strings.ToLower(typ.Field(i).Name)] = true
	}
	for _, id := range section.Order() {
		if id != section.Images && !fields[string(id)] {
			t.Errorf("the section %s has no flag on a VM's Sections", id)
		}
	}
	if len(fields) != len(section.Order())-1 {
		t.Errorf("Sections holds %d flags; want one per section but Images", len(fields))
	}
	server := machines.SectionsOf(machines.VM{Lifetime: machines.AlwaysOn})
	if !server.Overview || !server.Activity || !server.Ports || !server.SSH || server.Apps || server.Settings || server.Disk || server.Accounts || server.Background || server.Awake {
		t.Errorf("a server's sections: %+v; want Overview, Activity, Ports and SSH", server)
	}
	workspace := machines.SectionsOf(machines.VM{Lifetime: machines.Ephemeral})
	want := section.For(machines.Ephemeral.Kind())
	got := reflect.ValueOf(workspace)
	for i := range typ.NumField() {
		id := section.ID(strings.ToLower(typ.Field(i).Name))
		if got.Field(i).Bool() != slices.Contains(want, id) {
			t.Errorf("a workspace's %s is %v; section.For says otherwise", id, got.Field(i).Bool())
		}
	}
}

func TestAVMNotCreatedHasItsOverviewAlone(t *testing.T) {
	for _, lifetime := range []machines.Lifetime{machines.AlwaysOn, machines.Persistent, machines.Ephemeral} {
		vm := machines.VM{Lifetime: lifetime, State: machines.Stopped, Birth: &machines.Birth{Failed: true, Step: machines.BornBooting}}
		if got, want := machines.SectionsOf(vm), (machines.Sections{Overview: true}); got != want {
			t.Errorf("a %v not created: %+v; want its Overview alone", lifetime, got)
		}
	}
}

func TestAMacNotSeenIsNeverSaidDown(t *testing.T) {
	now := time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)
	silent := machines.MacView{Name: "mac-mini", Answered: now.Add(-12 * time.Minute)}
	got := machines.UnseenVerdict(silent, now)
	want := verdict.Verdict{Tone: verdict.Warn, Sentence: "mac-mini has not been seen for 12 min.", Line: machines.LineNothingProvesDown}
	if got == nil || *got != want {
		t.Errorf("a Mac silent 12 min: %+v; want %+v", got, want)
	}
	silent.Asleep = true
	if got := machines.UnseenVerdict(silent, now); got == nil || got.Sentence != "mac-mini is asleep." || got.Tone != verdict.Plain {
		t.Errorf("a Mac asleep: %+v", got)
	}
	if got := machines.UnseenVerdict(machines.MacView{Name: "mac-mini"}, now); got == nil || got.Sentence != "mac-mini has never answered." {
		t.Errorf("a Mac never heard: %+v", got)
	}
	if got := machines.UnseenVerdict(machines.MacView{Name: "mac-mini", Answered: now.Add(-3 * time.Hour)}, now); got == nil || got.Sentence != "mac-mini has not been seen for 3 h." {
		t.Errorf("a Mac silent 3 h: %+v", got)
	}
	if machines.UnseenVerdict(machines.MacView{Name: "mac-mini", Reachable: true}, now) != nil {
		t.Errorf("a Mac that answers has a not-seen verdict")
	}
}

func TestAShareSaysWhatOfItsMemoryTheVMUses(t *testing.T) {
	mac := machines.MacView{Hardware: coremachine.Hardware{Memory: 32 << 30}, VMs: []machines.VM{
		{Name: "linux-1", Lifetime: machines.Persistent, State: machines.Running, Resources: machines.Resources{MemoryBytes: 10 << 30}, UsedBytes: 3 << 30},
		{Name: "linux-2", Lifetime: machines.Persistent, State: machines.Running, Resources: machines.Resources{MemoryBytes: 4 << 30}, UsedBytes: 5 << 30},
	}}
	c, ok := machines.CapacityOf(mac)
	if !ok || len(c.VMs) != 2 {
		t.Fatalf("capacity: %+v", c)
	}
	if c.VMs[0].UsedBytes != 3<<30 || c.VMs[0].Bytes != 10<<30 {
		t.Errorf("linux-1 holds 10 GB and uses 3: %+v", c.VMs[0])
	}
	if c.VMs[1].UsedBytes != 4<<30 {
		t.Errorf("a VM's use is at most what it holds: %+v", c.VMs[1])
	}
}

func TestAVMsHeaderOffersWhatItsStateAllows(t *testing.T) {
	for name, c := range map[string]struct {
		vm   machines.VM
		want []machines.Gesture
	}{
		"a running server":      {machines.VM{Lifetime: machines.AlwaysOn, State: machines.Running}, []machines.Gesture{machines.GestureStop, machines.GestureRestart}},
		"a running workspace":   {machines.VM{Lifetime: machines.Persistent, State: machines.Running}, []machines.Gesture{machines.GestureStop}},
		"an ephemeral one":      {machines.VM{Lifetime: machines.Ephemeral, State: machines.Running}, []machines.Gesture{machines.GestureStop}},
		"a stopped server":      {machines.VM{Lifetime: machines.AlwaysOn, State: machines.Stopped}, []machines.Gesture{machines.GestureStart}},
		"a stopped workspace":   {machines.VM{Lifetime: machines.Persistent, State: machines.Stopped}, []machines.Gesture{machines.GestureStart}},
		"a suspended workspace": {machines.VM{Lifetime: machines.Persistent, State: machines.Suspended}, []machines.Gesture{machines.GestureStart}},
		"one being created":     {machines.VM{Lifetime: machines.Persistent, State: machines.Creating}, nil},
		"one not seen":          {machines.VM{Lifetime: machines.AlwaysOn, State: machines.UnknownState}, nil},
	} {
		if got := machines.GesturesOf(c.vm); !slices.Equal(got, c.want) {
			t.Errorf("%s offers %v, want %v", name, got, c.want)
		}
	}
	f := machines.Judged(machines.AllMachines{Macs: []machines.MacView{{VMs: []machines.VM{{Lifetime: machines.AlwaysOn, State: machines.Running}}}}})
	if got := f.Macs[0].VMs[0].Gestures; len(got) != 2 {
		t.Errorf("machines.list leaves a server's gestures unjudged: %v", got)
	}
}
