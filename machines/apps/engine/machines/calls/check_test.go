package calls_test

import (
	"strings"
	"testing"
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	machinesfake "github.com/jterrazz/studio-station/apps/engine/machines/adapter/fake"
	machinescontract "github.com/jterrazz/studio-station/apps/engine/machines/contract"
	setupcontract "github.com/jterrazz/studio-station/apps/engine/setup/contract"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	machines "github.com/jterrazz/studio-station/apps/engine/machines/calls"
)

func TestAListedServersCheckHoldsItsTokenWithheldAndItsRecordAsGiven(t *testing.T) {
	ctx := t.Context()
	check := "https://svc.example/health?token=hunter2hunter2"
	vms := machinesfake.NewStore()
	if err := vms.Put(ctx, model.Record{Name: "api", Lifetime: model.AlwaysOn, Check: check}); err != nil {
		t.Fatal(err)
	}
	prefs := memoryPrefs{}
	p := machines.Prefs{Name: setupcontract.Name(prefs), Macs: machinescontract.Macs(prefs), VMUser: machinescontract.VMUser(prefs)}
	if err := p.Name.Set("laptop"); err != nil {
		t.Fatal(err)
	}
	svc := machines.New(&stillMac{at: time.Unix(0, 0)}, nil, p, vms, noTailnet{}, nil, nil, nil, machines.Options{ThisMachine: catalog.Mac})
	all, err := svc.AllMachinesAsItFills(ctx, false, func(model.AllMachines) {})
	if err != nil || len(all.Macs) == 0 || len(all.Macs[0].VMs) != 1 {
		t.Fatalf("the fleet = %+v, %v; want this Mac with its one server", all, err)
	}
	if got := all.Macs[0].VMs[0].Check; got != "https://svc.example/health?token=withheld" {
		t.Errorf("the listed check = %q; want its token withheld", got)
	}
	if rec, err := vms.Get(ctx, "api"); err != nil || rec.Check != check || strings.Contains(check, "withheld") {
		t.Errorf("the record's check = %q, %v; want it as given, for the check to run", rec.Check, err)
	}
}
