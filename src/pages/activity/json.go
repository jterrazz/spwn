package activity

import (
	"time"

	"github.com/jterrazz/studio-station/src/activity"
	"github.com/jterrazz/studio-station/src/core/finding"
)

type MachineJSON struct {
	Name    string  `json:"name"`
	Label   string  `json:"label"`
	Kind    string  `json:"kind"`
	On      *string `json:"on"`
	ThisMac bool    `json:"this_mac"`
}

type CPUJSON struct {
	Busy *float64   `json:"busy_percent"`
	CPUs int        `json:"cpus"`
	Load [3]float64 `json:"load"`
}

type MemoryJSON struct {
	Used     int64   `json:"used_bytes"`
	Total    int64   `json:"total_bytes"`
	Pressure *string `json:"pressure"`
	Swap     int64   `json:"swap_used_bytes"`
}

type DiskJSON struct {
	Free       *int64   `json:"free_bytes"`
	Total      *int64   `json:"total_bytes"`
	Read       *float64 `json:"read_bytes_per_s"`
	Write      *float64 `json:"write_bytes_per_s"`
	ReadSince  uint64   `json:"read_since_boot_bytes"`
	WriteSince uint64   `json:"written_since_boot_bytes"`
}

type NetworkJSON struct {
	In            *float64 `json:"in_bytes_per_s"`
	Out           *float64 `json:"out_bytes_per_s"`
	ReceivedSince uint64   `json:"received_since_boot_bytes"`
	SentSince     uint64   `json:"sent_since_boot_bytes"`
}

type ThermalJSON struct {
	Level   string   `json:"level"`
	Celsius *float64 `json:"cpu_celsius"`
}

type BatteryJSON struct {
	Percent int    `json:"percent"`
	State   string `json:"state"`
	Health  *int   `json:"health_percent"`
	Cycles  *int   `json:"cycles"`
}

type ProcessJSON struct {
	Name   string   `json:"name"`
	PID    int      `json:"pid"`
	VM     *string  `json:"vm"`
	CPU    *float64 `json:"cpu_percent"`
	Memory *int64   `json:"memory_bytes"`
}

type TopJSON struct {
	ByCPU    []ProcessJSON `json:"by_cpu"`
	ByMemory []ProcessJSON `json:"by_memory"`
}

type RunJSON struct {
	CPU    float64 `json:"cpu_percent"`
	Memory int64   `json:"resident_bytes"`
	Mac    string  `json:"mac"`
}

type PointJSON struct {
	At      time.Time `json:"at"`
	CPU     *float64  `json:"cpu_percent"`
	MemUsed int64     `json:"memory_used_bytes"`
	Swap    int64     `json:"swap_used_bytes"`
	Free    int64     `json:"disk_free_bytes"`
	Read    float64   `json:"read_bytes_per_s"`
	Write   float64   `json:"write_bytes_per_s"`
	In      float64   `json:"in_bytes_per_s"`
	Out     float64   `json:"out_bytes_per_s"`
}

type HistoryJSON struct {
	Step   int         `json:"step_s"`
	Points []PointJSON `json:"points"`
}

type Body struct {
	Source   string         `json:"source"`
	State    string         `json:"state"`
	Window   *float64       `json:"window_s"`
	Since    *time.Time     `json:"history_since_at"`
	CPU      *CPUJSON       `json:"cpu"`
	Memory   *MemoryJSON    `json:"memory"`
	Disk     *DiskJSON      `json:"disk"`
	Network  *NetworkJSON   `json:"network"`
	Thermal  *ThermalJSON   `json:"thermal"`
	Battery  *BatteryJSON   `json:"battery"`
	Up       *int64         `json:"up_s"`
	Boot     *time.Time     `json:"boot_at"`
	Top      TopJSON        `json:"top"`
	Run      *RunJSON       `json:"run"`
	History  *HistoryJSON   `json:"history"`
	Findings []finding.JSON `json:"findings"`
}

type ReadJSON struct {
	Schema      string      `json:"schema"`
	GeneratedAt string      `json:"generated_at"`
	Machine     MachineJSON `json:"machine"`
	Body
}

type EveryJSON struct {
	Machine MachineJSON `json:"machine"`
	Body
	Error *ErrorJSON `json:"error"`
}

type ErrorJSON struct {
	Type    string  `json:"type"`
	Code    *string `json:"code"`
	Message string  `json:"message"`
}

type AllMachinesJSON struct {
	Schema      string         `json:"schema"`
	GeneratedAt string         `json:"generated_at"`
	Findings    []finding.JSON `json:"findings"`
	Machines    []EveryJSON    `json:"machines"`
}

func ptr[T any](v T) *T { return &v }

func machineOf(v View) MachineJSON {
	m := MachineJSON{Name: v.Name, Label: v.Label, Kind: v.Kind, ThisMac: v.ThisMac}
	if v.Mac != "" {
		m.On = ptr(v.Mac)
	}
	return m
}

func Contract(v View, now time.Time) ReadJSON {
	return ReadJSON{Schema: Schema, GeneratedAt: now.UTC().Format(time.RFC3339), Machine: machineOf(v), Body: body(v)}
}

func findings(v View) []finding.JSON {
	fs := seen(v)
	out := make([]finding.JSON, 0, len(fs))
	for _, f := range fs {
		out = append(out, finding.JSON{Severity: f.Severity, Section: f.Section, Subject: f.Subject, Text: f.Text, Fix: finding.Fix{Command: f.Fix}.JSON()})
	}
	return out
}

func processes(ps []activity.Process, byCPU bool) []ProcessJSON {
	out := make([]ProcessJSON, 0, len(ps))
	for _, p := range ps {
		j := ProcessJSON{Name: p.Name, PID: p.PID}
		if p.VM != "" {
			j.VM = ptr(p.VM)
		}
		if byCPU {
			j.CPU = ptr(p.Value)
		} else {
			j.Memory = ptr(int64(p.Value))
		}
		out = append(out, j)
	}
	return out
}

func body(v View) Body {
	// invariant: bytes, bytes a second and percentages 0-100; a value not
	// read is null, never a zero: a VM's thermal and battery are its Mac's.
	r := v.Report
	s := r.Last.Sample
	b := Body{Source: string(r.Source), State: "answering", Top: TopJSON{ByCPU: []ProcessJSON{}, ByMemory: []ProcessJSON{}}, Findings: findings(v)}
	switch {
	case v.Stopped:
		b.State, b.Source = "stopped", ""
		return b
	case v.Silent != "":
		b.State, b.Source = "does-not-answer", ""
		return b
	case r.Silent:
		b.State = "no-answer"
		return b
	}
	if !r.Since.IsZero() {
		b.Since = ptr(r.Since.UTC())
	}
	if run := r.Last.Run; run != nil {
		b.Run = &RunJSON{CPU: run.CPU, Memory: run.Memory, Mac: v.On}
		if !run.Started.IsZero() {
			b.Up = ptr(int64(s.At.Sub(run.Started).Seconds()))
		}
		return b
	}
	rated := r.Window > 0 && counted(s)
	rate := func(x float64) *float64 {
		if !rated {
			return nil
		}
		return ptr(x)
	}
	if rated {
		b.Window = ptr(r.Window.Seconds())
	}
	b.CPU = &CPUJSON{Busy: rate(r.Rates.CPU), CPUs: s.CPUs, Load: s.Load}
	b.Memory = &MemoryJSON{Used: s.MemUsed, Total: s.MemTotal, Swap: s.SwapUsed}
	if s.Pressure != "" {
		b.Memory.Pressure = ptr(string(s.Pressure))
	}
	b.Disk = &DiskJSON{Read: rate(r.Rates.DiskRead), Write: rate(r.Rates.DiskWrite), ReadSince: s.DiskRead, WriteSince: s.DiskWritten}
	if s.DiskTotal > 0 {
		b.Disk.Free, b.Disk.Total = ptr(s.DiskFree), ptr(s.DiskTotal)
	}
	b.Network = &NetworkJSON{In: rate(r.Rates.NetIn), Out: rate(r.Rates.NetOut), ReceivedSince: s.NetIn, SentSince: s.NetOut}
	if s.Thermal != activity.ThermalUnknown && v.Kind == computer {
		b.Thermal = &ThermalJSON{Level: lowerFirst(activity.Thermal(s.Thermal))}
	}
	if bat := s.Battery; bat != nil && v.Kind == computer {
		b.Battery = &BatteryJSON{Percent: bat.Percent, State: activity.BatteryState(*bat)}
		if bat.Health > 0 {
			b.Battery.Health = ptr(bat.Health)
		}
		if bat.Cycles > 0 {
			b.Battery.Cycles = ptr(bat.Cycles)
		}
	}
	if !s.Boot.IsZero() {
		b.Up, b.Boot = ptr(int64(s.At.Sub(s.Boot).Seconds())), ptr(s.Boot.UTC())
	}
	b.Top = TopJSON{ByCPU: processes(r.Last.Top.CPU, true), ByMemory: processes(r.Last.Top.Memory, false)}
	if r.History != nil {
		b.History = history(r.History)
	}
	return b
}

func history(points []activity.Sample) *HistoryJSON {
	h := &HistoryJSON{Step: int(activity.CoarseEvery.Seconds()), Points: []PointJSON{}}
	for i, x := range points {
		p := PointJSON{At: x.At.UTC(), MemUsed: x.MemUsed, Swap: x.SwapUsed, Free: x.DiskFree}
		if i > 0 {
			r := activity.Between(points[i-1], x)
			p.CPU, p.Read, p.Write, p.In, p.Out = ptr(r.CPU), r.DiskRead, r.DiskWrite, r.NetIn, r.NetOut
		}
		h.Points = append(h.Points, p)
	}
	return h
}
