package darwin

import "time"

type Ticks struct {
	User   uint64 `json:"user"`
	System uint64 `json:"system"`
	Idle   uint64 `json:"idle"`
	Nice   uint64 `json:"nice"`
}

type Pages struct {
	// value: host_statistics64's page counts, sized by Size bytes each.
	Size       uint64 `json:"size"`
	Internal   uint64 `json:"internal"`
	Purgeable  uint64 `json:"purgeable"`
	Wired      uint64 `json:"wired"`
	Compressor uint64 `json:"compressor"`
}

type BlockDriver struct {
	// value: an IOBlockStorageDriver and its device's Protocol Characteristics.
	Media        string `json:"media"`
	Interconnect string `json:"interconnect"`
	Location     string `json:"location"`
	Read         uint64 `json:"read"`
	Written      uint64 `json:"written"`
}

type Link struct {
	// value: an interface and its byte counters, net.link.generic's IFMIB_IFDATA.
	Name string `json:"name"`
	In   uint64 `json:"in"`
	Out  uint64 `json:"out"`
}

type Counters struct {
	// value: the logical CPUs, when no per-CPU ticks are read (a build without cgo).
	CPUCount int           `json:"cpu_count,omitempty"`
	CPUs     []Ticks       `json:"cpus"`
	Load     [3]float64    `json:"load"`
	Pages    Pages         `json:"pages"`
	SwapUsed uint64        `json:"swap_used"`
	Pressure int           `json:"pressure"`
	Memsize  uint64        `json:"memsize"`
	Up       time.Duration `json:"up"`
	Disks    []BlockDriver `json:"disks"`
	Links    []Link        `json:"links"`
}

type Proc struct {
	PID      int           `json:"pid"`
	Name     string        `json:"name"`
	CPU      time.Duration `json:"cpu"`
	Resident uint64        `json:"resident"`
	// value: when the process started; read for one process by its pid,
	// zero in a whole table.
	Started time.Time `json:"started,omitzero"`
}

type Procs struct {
	// value: the time of the reading on the kernel's own clock, so two
	// readings tell the time between them without the wall clock.
	At   time.Duration `json:"at"`
	List []Proc        `json:"list"`
}

func (c Counters) Cores() int {
	// value: the logical CPUs: the ticks read when there are some, else the count a
	// build without cgo reads.
	return max(len(c.CPUs), c.CPUCount)
}
