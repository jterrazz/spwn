package calls_test

import (
	"context"
	"slices"
	"testing"
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	machinesfake "github.com/jterrazz/studio-station/apps/engine/machines/adapter/fake"
	machinescontract "github.com/jterrazz/studio-station/apps/engine/machines/contract"
	setupcontract "github.com/jterrazz/studio-station/apps/engine/setup/contract"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	machines "github.com/jterrazz/studio-station/apps/engine/machines/calls"
)

func allMachinesAsking(t *testing.T, ask func(ctx context.Context, macs []string, answered func(string, model.MacView))) *machines.Machinist {
	t.Helper()
	return allMachinesRemembering(t, map[string]model.MacRecord{"laptop": {}, "mac-mini": {}, "studio": {}}, ask)
}

func allMachinesRemembering(t *testing.T, macs map[string]model.MacRecord, ask func(ctx context.Context, macs []string, answered func(string, model.MacView))) *machines.Machinist {
	t.Helper()
	h := &stillMac{at: time.Unix(0, 0)}
	prefs := memoryPrefs{}
	p := machines.Prefs{Name: setupcontract.Name(prefs), Macs: machinescontract.Macs(prefs), VMUser: machinescontract.VMUser(prefs)}
	if err := p.Name.Set("laptop"); err != nil {
		t.Fatal(err)
	}
	if err := p.Macs.Set(macs); err != nil {
		t.Fatal(err)
	}
	return machines.New(h, nil, p, machinesfake.NewStore(), noTailnet{}, nil, nil, nil, machines.Options{ThisMachine: catalog.Mac, Ask: ask})
}

func names(f model.AllMachines) []string {
	out := make([]string, 0, len(f.Macs))
	for _, h := range f.Macs {
		out = append(out, h.Name)
	}
	return out
}

func TestTheAllMachinesFillsARowAsEachMacAnswersAndASilentOneEndsUnreachable(t *testing.T) {
	svc := allMachinesAsking(t, func(_ context.Context, _ []string, answered func(string, model.MacView)) {
		answered("studio", model.MacView{Name: "studio"})
	})
	var seen [][]string
	allMachines, err := svc.AllMachinesAsItFills(t.Context(), false, func(f model.AllMachines) { seen = append(seen, names(f)) })
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"laptop"}, {"laptop", "studio"}}
	if !slices.EqualFunc(seen, want, slices.Equal[[]string]) {
		t.Errorf("the rows as they filled: %q, want %q: this Mac at once, the studio when it answered, the silent mini not yet", seen, want)
	}
	got := names(allMachines)
	if !slices.Equal(got, []string{"laptop", "mac-mini", "studio"}) || allMachines.Macs[1].Reachable || !allMachines.Macs[2].Reachable {
		t.Errorf("the fleet at the end: %q, mini reachable %v; want every Mac, in name order, the silent mini unreachable", got, allMachines.Macs[1].Reachable)
	}
}

func TestASilentMacsVMKeepsTheKindItLastAnsweredWith(t *testing.T) {
	answering := true
	svc := allMachinesAsking(t, func(_ context.Context, _ []string, answered func(string, model.MacView)) {
		if answering {
			answered("studio", model.MacView{Name: "studio", VMs: []model.VM{{Name: "xcode", Kind: catalog.MacOS, Lifetime: model.Persistent}}})
		}
	})
	heard, err := svc.AllMachines(t.Context(), false)
	if err != nil {
		t.Fatal(err)
	}
	if err = svc.Remember(t.Context(), heard); err != nil {
		t.Fatal(err)
	}
	answering = false
	silent, err := svc.AllMachines(t.Context(), false)
	if err != nil {
		t.Fatal(err)
	}
	studio := silent.Macs[2]
	if studio.Reachable || len(studio.VMs) != 1 || studio.VMs[0].Kind != catalog.MacOS {
		t.Errorf("the silent studio: reachable %v, VMs %+v; want xcode kept a macos VM", studio.Reachable, studio.VMs)
	}
}

func TestAVMRememberedWithoutItsKindIsLeftOutUntilItsMacAnswers(t *testing.T) {
	answering := false
	remembered := model.MacRecord{VMs: []model.SeenVM{{Name: "old", Lifetime: model.Persistent, CPUs: 2, MemoryBytes: 4 << 30}}}
	svc := allMachinesRemembering(t, map[string]model.MacRecord{"laptop": {}, "studio": remembered}, func(_ context.Context, _ []string, answered func(string, model.MacView)) {
		if answering {
			answered("studio", model.MacView{Name: "studio", VMs: []model.VM{{Name: "old", Kind: catalog.Linux, Lifetime: model.Persistent}}})
		}
	})
	silent, err := svc.AllMachines(t.Context(), false)
	if err != nil {
		t.Fatal(err)
	}
	if vms := silent.Macs[1].VMs; len(vms) != 0 {
		t.Errorf("the silent studio's VMs: %+v; want none, its one VM remembered without a kind", vms)
	}
	answering = true
	heard, err := svc.AllMachines(t.Context(), false)
	if err != nil {
		t.Fatal(err)
	}
	if vms := heard.Macs[1].VMs; len(vms) != 1 || vms[0].Kind != catalog.Linux {
		t.Errorf("the studio once it answers: %+v; want old back, a linux VM", vms)
	}
}
