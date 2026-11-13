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

// value: the least memory a VM is made with, in bytes: a create asking
// less is refused, and a Mac with less free takes no VM.
const (
	MinMemoryBytes = 2 << 30
)

type Preset string

// value: the sizes a new VM is offered in one click, smallest first.
const (
	PresetSmall     Preset = "small"
	PresetWorkspace Preset = "workspace"
	PresetGate      Preset = "gate"
)

func (Preset) Enum() []string {
	return []string{string(PresetSmall), string(PresetWorkspace), string(PresetGate)}
}

func (p Preset) Word() string { return string(p) }

func (p Preset) MemoryBytes() int64 {
	// value: the preset's memory, in bytes (owner, 2026-11-13).
	switch p {
	case PresetSmall:
		return 4 << 30
	case PresetWorkspace:
		return 8 << 30
	case PresetGate:
		return 12 << 30
	}
	return 0
}

func Presets() []Preset { return []Preset{PresetSmall, PresetWorkspace, PresetGate} }

type Fit string

// value: whether a size fits a Mac, by Fits.
const (
	FitsIn Fit = "fits"
	NoRoom Fit = "no-room"
)

func (Fit) Enum() []string { return []string{string(FitsIn), string(NoRoom)} }

func FitOf(size Resources, c Capacity) Fit {
	if Fits(size, c) {
		return FitsIn
	}
	return NoRoom
}

type PresetOn struct {
	Preset      Preset
	MemoryBytes int64
	Fit         Fit
}

type RoomFor string

// value: the largest preset a Mac still fits, or none.
const (
	RoomForSmall     RoomFor = RoomFor(PresetSmall)
	RoomForWorkspace RoomFor = RoomFor(PresetWorkspace)
	RoomForGate      RoomFor = RoomFor(PresetGate)
	RoomForNone      RoomFor = "none"
)

func (RoomFor) Enum() []string {
	return []string{string(RoomForSmall), string(RoomForWorkspace), string(RoomForGate), string(RoomForNone)}
}

type Room struct {
	// value: the Mac, by its fleet name, and its capacity.
	Mac      string
	Capacity Capacity
	// value: each preset, judged on this Mac by Fits, and the largest that
	// fits.
	Presets []PresetOn
	RoomFor RoomFor
}

func RoomOf(mac string, c Capacity) Room {
	r := Room{Mac: mac, Capacity: c, RoomFor: RoomForNone}
	for _, p := range Presets() {
		fit := FitOf(Resources{MemoryBytes: p.MemoryBytes()}, c)
		r.Presets = append(r.Presets, PresetOn{Preset: p, MemoryBytes: p.MemoryBytes(), Fit: fit})
		if fit == FitsIn {
			r.RoomFor = RoomFor(p)
		}
	}
	return r
}

func Fits(size Resources, c Capacity) bool {
	// invariant: memory alone judges a size, a create's refusal and every
	// room a client draws alike, so the largest size that fits is Free.
	return size.MemoryBytes <= c.Free
}

func Rooms(f AllMachines) []Room {
	// value: the room of each Mac of the fleet, none for one that never
	// said its memory.
	out := make([]Room, 0, len(f.Macs))
	for _, h := range f.Macs {
		if c, ok := CapacityOf(h); ok {
			out = append(out, RoomOf(h.Name, c))
		}
	}
	return out
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
	// value: the Mac a new VM fits best on, of those that Takes one, the
	// one with the most free memory (This Mac first on a tie); empty when
	// none takes one.
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
		c, _ := CapacityOf(h)
		if !Takes(h) || c.Free <= room {
			continue
		}
		best, room = h.Name, c.Free
	}
	return best
}

func Takes(h MacView) bool {
	// value: a new VM may go on that Mac now: it answers and is awake,
	// says its memory, and has at least MinMemoryBytes free.
	c, ok := CapacityOf(h)
	return h.Reachable && !h.Asleep && ok && c.Free >= MinMemoryBytes
}

func Preselected(f AllMachines, asked string) string {
	// value: the Mac a new VM is offered on: the one asked for when it
	// takes one, else Roomiest, else This Mac.
	for _, h := range f.Macs {
		if h.Name == asked && Takes(h) {
			return h.Name
		}
	}
	if r := Roomiest(f); r != "" {
		return r
	}
	for _, h := range f.Macs {
		if h.ThisMac {
			return h.Name
		}
	}
	return ""
}
