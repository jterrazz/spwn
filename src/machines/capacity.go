package machines

import "slices"

const KeptForMacOS = 4 << 30

type Capacity struct {
	// value: SPEC B5.12: a Mac's memory, what macOS keeps of it, what each
	// VM reserves and what a new VM may take, in bytes.
	Total, MacOS, Free int64
	VMs                []Share
}

type Share struct {
	Name  string
	Bytes int64
}

func CapacityOf(h MacView) (Capacity, bool) {
	// invariant: a VM reserves its memory while it runs or is being made,
	// and a server its record wants running keeps its share while it is
	// down, so the free figure never jumps when it falls and comes back;
	// unknown when the Mac never said what it has.
	if h.Hardware.Memory <= 0 {
		return Capacity{}, false
	}
	c := Capacity{Total: h.Hardware.Memory, MacOS: KeptForMacOS}
	free := c.Total - c.MacOS
	for _, vm := range Ordered(h.VMs) {
		if reserves(vm) {
			c.VMs = append(c.VMs, Share{Name: vm.Name, Bytes: vm.Resources.MemoryBytes})
			free -= vm.Resources.MemoryBytes
		}
	}
	c.Free = max(free, 0)
	return c, true
}

func reserves(vm VM) bool {
	switch {
	case vm.State == Running, vm.State == Creating:
		return true
	case vm.Lifetime == AlwaysOn:
		return vm.Want == WantRunning || vm.State == UnknownState
	}
	return false
}

func Roomiest(f AllMachines) string {
	// value: the awake Mac a new VM fits best on, the one with the most
	// free memory (This Mac first on a tie); empty when none says.
	best, room := "", int64(-1)
	macs := slices.Clone(f.Macs)
	slices.SortStableFunc(macs, func(a, b MacView) int {
		if a.ThisMac != b.ThisMac {
			if a.ThisMac {
				return -1
			}
			return 1
		}
		return 0
	})
	for _, h := range macs {
		c, ok := CapacityOf(h)
		if !h.Reachable || !ok || c.Free <= room {
			continue
		}
		best, room = h.Name, c.Free
	}
	return best
}
