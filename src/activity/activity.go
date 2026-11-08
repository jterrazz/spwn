package activity

import (
	"strconv"
	"time"
)

type Pressure string

const (
	PressureNormal   Pressure = "normal"
	PressureWarn     Pressure = "warn"
	PressureCritical Pressure = "critical"
)

const (
	ThermalUnknown  = -1
	ThermalSerious  = 2
	ThermalCritical = 3
)

type Battery struct {
	Percent   int  `json:"percent"`
	Charging  bool `json:"charging"`
	OnBattery bool `json:"on_battery"`
	// value: the battery's health in percent and its cycle count; 0 unread.
	Health int `json:"health,omitempty"`
	Cycles int `json:"cycles,omitempty"`
}

type Sample struct {
	At time.Time `json:"at"`
	// value: cumulative CPU ticks of the whole machine, busy and in all.
	CPUBusy  uint64     `json:"cpu_busy"`
	CPUTotal uint64     `json:"cpu_total"`
	CPUs     int        `json:"cpus"`
	Load     [3]float64 `json:"load"`
	// value: binary bytes; MemUsed is what Activity Monitor calls Memory Used.
	MemUsed  int64 `json:"mem_used"`
	MemTotal int64 `json:"mem_total"`
	// invariant: "" when the machine did not say.
	Pressure Pressure `json:"pressure,omitempty"`
	SwapUsed int64    `json:"swap_used"`
	// value: the kernel's counters since boot, of whole physical disks and
	// real links only: station keeps no total of its own.
	DiskRead    uint64 `json:"disk_read"`
	DiskWritten uint64 `json:"disk_written"`
	NetIn       uint64 `json:"net_in"`
	NetOut      uint64 `json:"net_out"`
	// value: the free space the Disk family read; 0 when not composed in.
	DiskFree  int64 `json:"disk_free,omitempty"`
	DiskTotal int64 `json:"disk_total,omitempty"`
	// value: the thermal pressure level Power read, 0 nominal to 4;
	// ThermalUnknown when not composed in.
	Thermal int `json:"thermal"`
	// invariant: nil on a machine without a battery, or when not composed in.
	Battery *Battery `json:"battery,omitempty"`
	// invariant: zero when the machine did not say when it booted.
	Boot time.Time `json:"boot"`
}

type Process struct {
	PID  int    `json:"pid"`
	Name string `json:"name"`
	// value: a CPU share of the whole machine (0 to 100) in a list by CPU,
	// resident bytes in a list by memory.
	Value float64 `json:"value"`
	// value: the VM whose Virtualization process this is, on its Mac.
	VM string `json:"vm,omitempty"`
}

const Top = 5

type Busiest struct {
	// invariant: never nil: a list nothing answered is empty; at most Top
	// each, busiest first.
	CPU    []Process `json:"cpu"`
	Memory []Process `json:"memory"`
}

type Usage struct {
	// value: one process's CPU share of the whole machine, its resident
	// bytes and when it started (zero when the kernel does not say).
	CPU     float64   `json:"cpu"`
	Memory  int64     `json:"memory"`
	Started time.Time `json:"started,omitzero"`
}

func Uptime(d time.Duration) string {
	// value: the uptime the owner reads: "22d 14h", "3h 5m", "12m".
	minutes := int(d.Minutes())
	days, hours := minutes/(24*60), minutes/60%24
	switch {
	case days > 0:
		return strconv.Itoa(days) + "d " + strconv.Itoa(hours) + "h"
	case hours > 0:
		return strconv.Itoa(hours) + "h " + strconv.Itoa(minutes%60) + "m"
	}
	return strconv.Itoa(minutes) + "m"
}

type Reading struct {
	Sample Sample  `json:"sample"`
	Top    Busiest `json:"top"`
	// value: a VM read from its Mac while its guest does not answer: its
	// process's usage there, and nothing of the VM's own counters.
	Run *Usage `json:"run,omitempty"`
	// value: on a Mac, each VM it runs as this Mac measures it, from its
	// Virtualization process: its CPU a share of the whole Mac (SPEC B5.3).
	VMs []VMUsage `json:"vms,omitempty"`
}

type VMUsage struct {
	VM string `json:"vm"`
	Usage
}

type VMRun struct {
	// value: a VM this Mac runs, and the process its guest runs in there
	// (Virtualization's, not its run's): 0 while none holds it yet.
	VM     string `json:"vm"`
	Holder int    `json:"holder"`
}
