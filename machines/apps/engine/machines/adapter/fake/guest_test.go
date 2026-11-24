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
		g, err := GuestsOf(w).Dial(t.Context(), contract.GuestTarget{VM: "dev"})
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
