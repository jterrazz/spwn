package machines

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

const gb = int64(1) << 30

func mini(vms ...VM) MacView {
	return MacView{Name: "mac-mini", Reachable: true, Hardware: machine.Hardware{Memory: 64 * gb}, VMs: vms}
}

func sized(vm VM, memory int64) VM {
	vm.Resources.MemoryBytes = memory
	return vm
}

func TestAMacsFreeMemoryIsWhatNoVMReserves(t *testing.T) {
	infra := sized(VM{Name: "infra", Lifetime: AlwaysOn, Want: WantRunning, State: Running}, 24*gb)
	linux := sized(VM{Name: "linux-3", Lifetime: Ephemeral, State: Running}, 8*gb)
	api := sized(VM{Name: "api", Lifetime: Persistent, Want: WantRunning, State: Stopped}, 8*gb)
	c, ok := CapacityOf(mini(linux, infra, api))
	if !ok {
		t.Fatal("a Mac that says its memory has a capacity")
	}
	if c.Total != 64*gb || c.MacOS != KeptForMacOS || c.Free != 28*gb {
		t.Errorf("capacity %+v: want 64 GB, 4 GB for macOS, 28 GB free", c)
	}
	if len(c.VMs) != 2 || c.VMs[0].Name != "infra" || c.VMs[1].Name != "linux-3" {
		t.Errorf("shares %+v: want infra then linux-3, a stopped workspace holding none", c.VMs)
	}
}

func TestADownServerKeepsItsShareSoTheFigureNeverJumps(t *testing.T) {
	up := sized(VM{Name: "infra", Lifetime: AlwaysOn, Want: WantRunning, State: Running}, 24*gb)
	for _, c := range []struct {
		name string
		vm   VM
		free int64
	}{
		{"running", up, 36 * gb},
		{"crashed, waiting to restart", func() VM { v := up; v.State = Stopped; return v }(), 36 * gb},
		{"broken", func() VM { v := up; v.State = Broken; return v }(), 36 * gb},
		{"on a Mac that does not answer", func() VM { v := up; v.State, v.Want = UnknownState, ""; return v }(), 36 * gb},
		{"stopped by its owner", func() VM { v := up; v.State, v.Want = Stopped, WantStopped; return v }(), 60 * gb},
	} {
		got, _ := CapacityOf(mini(c.vm))
		if got.Free != c.free {
			t.Errorf("a server %s: %d GB free, want %d", c.name, got.Free/gb, c.free/gb)
		}
	}
}

func TestAMacThatNeverSaidItsMemoryHasNoCapacity(t *testing.T) {
	if _, ok := CapacityOf(MacView{Name: "mac-mini", Reachable: true}); ok {
		t.Error("a capacity was invented for a Mac that never said its memory")
	}
	full, _ := CapacityOf(mini(sized(VM{Name: "big", State: Running}, 80*gb)))
	if full.Free != 0 {
		t.Errorf("free %d: a Mac given more than it has has none free, never less", full.Free)
	}
}

func TestTheRoomiestAwakeMacTakesANewVM(t *testing.T) {
	this := MacView{Name: "macbook-pro", ThisMac: true, Reachable: true, Hardware: machine.Hardware{Memory: 16 * gb}}
	silent := MacView{Name: "mac-studio", Hardware: machine.Hardware{Memory: 128 * gb}}
	for _, c := range []struct {
		name string
		f    AllMachines
		want string
	}{
		{"the Mac with the most free memory", AllMachines{Macs: []MacView{this, mini()}}, "mac-mini"},
		{"a silent Mac is never picked", AllMachines{Macs: []MacView{this, silent}}, "macbook-pro"},
		{"This Mac on a tie", AllMachines{Macs: []MacView{mini(), {Name: "mine", ThisMac: true, Reachable: true, Hardware: machine.Hardware{Memory: 64 * gb}}}}, "mine"},
		{"none when no Mac says its memory", AllMachines{Macs: []MacView{{Name: "macbook-pro", ThisMac: true, Reachable: true}}}, ""},
	} {
		if got := Roomiest(c.f); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

func TestASizeFitsTheFreeMemoryAndNotAByteMore(t *testing.T) {
	c, _ := CapacityOf(mini(sized(VM{Name: "infra", Lifetime: AlwaysOn, Want: WantRunning, State: Running}, 24*gb)))
	if !Fits(Resources{MemoryBytes: c.Free}, c) {
		t.Errorf("%d bytes on a Mac with %d free: refused, want the whole free figure to fit", c.Free, c.Free)
	}
	if Fits(Resources{MemoryBytes: c.Free + 1}, c) {
		t.Errorf("%d bytes on a Mac with %d free: fits, want one byte over refused", c.Free+1, c.Free)
	}
}

func TestEachMacThatSaidItsMemoryHasItsRoom(t *testing.T) {
	this := MacView{Name: "macbook-pro", ThisMac: true, Reachable: true, Hardware: machine.Hardware{Memory: 16 * gb}}
	never := MacView{Name: "mac-studio"}
	silent := MacView{Name: "mac-mini", Hardware: machine.Hardware{Memory: 64 * gb}, VMs: []VM{sized(VM{Name: "infra", Lifetime: AlwaysOn, State: UnknownState}, 24*gb)}}
	rooms := Rooms(AllMachines{Macs: []MacView{this, never, silent}})
	if len(rooms) != 2 || rooms[0].Mac != "macbook-pro" || rooms[1].Mac != "mac-mini" {
		t.Fatalf("rooms %+v: want macbook-pro then mac-mini, none for a Mac that never said its memory", rooms)
	}
	for i, h := range []MacView{this, silent} {
		if c, _ := CapacityOf(h); rooms[i].Capacity.Free != c.Free || len(rooms[i].Capacity.VMs) != len(c.VMs) {
			t.Errorf("%s's room %+v: want its capacity %+v", h.Name, rooms[i].Capacity, c)
		}
	}
}

func TestEachPresetIsJudgedByFitsAndTheRoomSaysTheLargestThatFits(t *testing.T) {
	for _, c := range []struct {
		free int64
		fits []Fit
		room RoomFor
	}{
		{12 * gb, []Fit{FitsIn, FitsIn, FitsIn}, RoomForGate},
		{8 * gb, []Fit{FitsIn, FitsIn, NoRoom}, RoomForWorkspace},
		{8*gb - 1, []Fit{FitsIn, NoRoom, NoRoom}, RoomForSmall},
		{3 * gb, []Fit{NoRoom, NoRoom, NoRoom}, RoomForNone},
	} {
		r := RoomOf("mac-mini", Capacity{Total: 64 * gb, MacOS: KeptForMacOS, Free: c.free})
		if len(r.Presets) != len(Presets()) {
			t.Fatalf("%d bytes free: presets %+v, want one per preset", c.free, r.Presets)
		}
		for i, p := range r.Presets {
			if p.Preset != Presets()[i] || p.MemoryBytes != p.Preset.MemoryBytes() || p.Fit != c.fits[i] {
				t.Errorf("%d bytes free: %+v, want %s judged %s", c.free, p, Presets()[i], c.fits[i])
			}
		}
		if r.RoomFor != c.room {
			t.Errorf("%d bytes free: room for %q, want %q", c.free, r.RoomFor, c.room)
		}
	}
}

func TestThePresetsAreFourEightAndTwelveGigabytes(t *testing.T) {
	want := map[Preset]int64{PresetSmall: 4 << 30, PresetWorkspace: 8 << 30, PresetGate: 12 << 30}
	for _, p := range Presets() {
		if p.MemoryBytes() != want[p] {
			t.Errorf("%s: %d bytes, want %d", p, p.MemoryBytes(), want[p])
		}
	}
	if MinMemoryBytes != 2<<30 {
		t.Errorf("the least memory a VM is made with is %d bytes, want 2 GiB", MinMemoryBytes)
	}
}

func TestTheRoomiestMacIsOneThatTakesAVM(t *testing.T) {
	this := MacView{Name: "macbook-pro", ThisMac: true, Reachable: true, Hardware: machine.Hardware{Memory: 16 * gb}}
	full := MacView{Name: "macbook-pro", ThisMac: true, Reachable: true, Hardware: machine.Hardware{Memory: 16 * gb}, VMs: []VM{sized(VM{Name: "big", State: Running}, 10*gb+gb/2)}}
	asleep := mini()
	asleep.Asleep = true
	for _, c := range []struct {
		name string
		f    AllMachines
		want string
	}{
		{"a Mac with less than the least memory a VM takes free", AllMachines{Macs: []MacView{full}}, ""},
		{"an asleep Mac, however roomy", AllMachines{Macs: []MacView{this, asleep}}, "macbook-pro"},
	} {
		if got := Roomiest(c.f); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

func TestANewVMIsOfferedOnTheMacAskedThenTheRoomiestThenThisMac(t *testing.T) {
	this := MacView{Name: "macbook-pro", ThisMac: true, Reachable: true, Hardware: machine.Hardware{Memory: 16 * gb}}
	silent := MacView{Name: "mac-studio", Hardware: machine.Hardware{Memory: 128 * gb}}
	full := MacView{Name: "macbook-pro", ThisMac: true, Reachable: true, Hardware: machine.Hardware{Memory: 4 * gb}}
	for _, c := range []struct {
		name  string
		f     AllMachines
		asked string
		want  string
	}{
		{"the Mac asked for, though another is roomier", AllMachines{Macs: []MacView{this, mini()}}, "macbook-pro", "macbook-pro"},
		{"the roomiest, when the Mac asked for is silent", AllMachines{Macs: []MacView{this, mini(), silent}}, "mac-studio", "mac-mini"},
		{"the roomiest, when none is asked for", AllMachines{Macs: []MacView{this, mini()}}, "", "mac-mini"},
		{"This Mac, when no Mac takes a VM", AllMachines{Macs: []MacView{full, silent}}, "", "macbook-pro"},
	} {
		if got := Preselected(c.f, c.asked); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}
