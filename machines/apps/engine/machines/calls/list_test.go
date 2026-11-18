package calls_test

import (
	"context"
	"testing"
	"time"

	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	machines "github.com/jterrazz/studio-station/apps/engine/machines/calls"
)

// value: a fleet read that answers this Mac first, then the whole fleet.
type fleet struct {
	machines.Machines
	first, all model.AllMachines
}

func (f fleet) AllMachinesAsItFills(_ context.Context, _ bool, sofar func(model.AllMachines)) (model.AllMachines, error) {
	sofar(f.first)
	return f.all, nil
}

func TestTheFleetReadSaysEachMacsRoomAndWhereANewVMGoes(t *testing.T) {
	this := model.MacView{Name: "macbook-pro", ThisMac: true, Reachable: true, Hardware: coremachine.Hardware{Memory: 16 << 30}}
	mini := model.MacView{Name: "mac-mini", Reachable: true, Hardware: coremachine.Hardware{Memory: 64 << 30}, VMs: []model.VM{
		{Name: "infra", Lifetime: model.AlwaysOn, Want: model.WantRunning, State: model.Running, Resources: model.Resources{MemoryBytes: 24 << 30}},
	}}
	never := model.MacView{Name: "mac-studio", Reachable: true}
	f := fleet{first: model.AllMachines{Macs: []model.MacView{this}}, all: model.AllMachines{Macs: []model.MacView{this, mini, never}}}
	bound, stop := context.WithTimeout(t.Context(), 5*time.Second)
	defer stop()
	updates, err := machines.List.Run(bound, machines.Deps{Machines: f}, machines.ListIn{Unwritten: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var got []machines.ListOut
	for {
		select {
		case u, open := <-updates:
			if !open {
				if len(got) != 2 {
					t.Fatalf("%d updates, want this Mac's then the fleet's", len(got))
				}
				if first := got[0]; len(first.Rooms) != 1 || first.Rooms[0].Mac != "macbook-pro" || first.Rooms[0].Capacity.Free != 12<<30 || first.Roomiest != "macbook-pro" {
					t.Errorf("the first update: rooms %+v, roomiest %q; want This Mac's 12 GB free, and This Mac", first.Rooms, first.Roomiest)
				}
				last := got[1]
				if len(last.Rooms) != 2 || last.Rooms[1].Mac != "mac-mini" || last.Rooms[1].Capacity.Free != 36<<30 || last.Roomiest != "mac-mini" {
					t.Errorf("the fleet: rooms %+v, roomiest %q; want the mini's 36 GB free beside This Mac's, none for the studio, and the mini", last.Rooms, last.Roomiest)
				}
				return
			}
			if u.Err != nil {
				t.Fatal(u.Err)
			}
			got = append(got, u.Value)
		case <-bound.Done():
			t.Fatal("the fleet read never ended")
		}
	}
}
