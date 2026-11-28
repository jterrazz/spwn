package fake

import (
	"io/fs"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestTheFakeGuestKeepsTheContract(t *testing.T) {
	contracttest.Guest(t, func(*testing.T) contracttest.GuestSubject {
		guests := NewGuests()
		vm := guests.Add("dev")
		vm.Serve(contracttest.GuestPort, contracttest.Greeting)
		return contracttest.GuestSubject{
			Dialer:   guests,
			Target:   func(*testing.T) contract.GuestTarget { return contract.GuestTarget{VM: "dev"} },
			Program:  func(line string, a contracttest.Answer) { vm.Program(line, Answer(a)) },
			Received: vm.Received,
			Read: func(t *testing.T, path string) ([]byte, fs.FileMode) {
				f, ok := vm.File(path)
				if !ok {
					t.Fatalf("no file %s in the guest", path)
				}
				return f.Data, f.Mode
			},
			Path: "/tmp/station",
		}
	})
}

func TestAFakeGuestPowersItsVMOffByItsSystemsPoweroffLine(t *testing.T) {
	for _, kind := range []catalog.MachineKind{catalog.Linux, catalog.MacOS} {
		w := &world.World{ThisMac: "this-mac"}
		w.Mac("this-mac").VMs["dev"] = &world.VM{PID: 1001, Guest: &world.Guest{}}
		g, err := GuestsOf(w, "this-mac").Dial(t.Context(), contract.GuestTarget{VM: "dev"})
		if err != nil {
			t.Fatal(err)
		}
		ran, err := g.Exec(t.Context(), contract.GuestCommand{Line: catalog.PoweroffLine(kind)})
		if err != nil || ran.Code != 0 {
			t.Fatalf("%s: the poweroff line answered %d, %v; want 0", kind, ran.Code, err)
		}
		if v := w.Macs["this-mac"].VMs["dev"]; v.Running() || v.Ended == nil || v.Ended.Reason != machines.GuestPoweroff {
			t.Errorf("%s: the VM is %+v after its poweroff line, want it ended by a guest poweroff", kind, v)
		}
	}
}

func TestAFakeGuestIsItsOwnMacsVMNeverAnotherMacsOfItsName(t *testing.T) {
	w := &world.World{ThisMac: "air"}
	w.Mac("macbook").VMs["dev"] = &world.VM{PID: 1001, Guest: &world.Guest{}}
	w.Mac("mini").VMs["dev"] = &world.VM{PID: 2001, Guest: &world.Guest{}}
	g, err := GuestsOf(w, "mini").Dial(t.Context(), contract.GuestTarget{VM: "dev"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Exec(t.Context(), contract.GuestCommand{Line: catalog.PoweroffLine(catalog.Linux)}); err != nil {
		t.Fatal(err)
	}
	if v := w.Macs["mini"].VMs["dev"]; v.Running() {
		t.Errorf("mini's dev is %+v after its poweroff line, want it ended", v)
	}
	if v := w.Macs["macbook"].VMs["dev"]; v.PID != 1001 || v.Ended != nil {
		t.Errorf("macbook's dev is %+v after mini's dev powered off, want it running as pid 1001", v)
	}
}
