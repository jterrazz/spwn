package activity

import (
	"encoding/json"
	"math"
	"time"
)

type unit int

const (
	// value: what a number an answer writes measures, and so the precision
	// its reader needs: a share or a load to 2 decimals, bytes and bytes a
	// second whole.
	share unit = iota
	whole
)

func rounded(x float64, u unit) float64 {
	// invariant: the one rule every float of an activity answer is written
	// by, so no answer carries the digits of a float's binary form.
	if u == share {
		return math.Round(x*100) / 100
	}
	return math.Round(x)
}

func (r Rates) MarshalJSON() ([]byte, error) {
	type plain Rates
	return json.Marshal(plain{
		CPU: rounded(r.CPU, share), DiskRead: rounded(r.DiskRead, whole), DiskWrite: rounded(r.DiskWrite, whole),
		NetIn: rounded(r.NetIn, whole), NetOut: rounded(r.NetOut, whole),
	})
}

func roundedBars(bars []Bucket, u unit) []Bucket {
	if bars == nil {
		return nil
	}
	out := make([]Bucket, len(bars))
	for i, b := range bars {
		b.Mean, b.Max = rounded(b.Mean, u), rounded(b.Max, u)
		out[i] = b
	}
	return out
}

func (w Chart) MarshalJSON() ([]byte, error) {
	type plain Chart
	w.Buckets = roundedBars(w.Buckets, share)
	return json.Marshal(plain(w))
}

func (m Measures) MarshalJSON() ([]byte, error) {
	type plain Measures
	return json.Marshal(plain{
		Every: m.Every, Memory: roundedBars(m.Memory, whole), Swap: roundedBars(m.Swap, whole),
		DiskRead: roundedBars(m.DiskRead, whole), DiskWrite: roundedBars(m.DiskWrite, whole),
		NetIn: roundedBars(m.NetIn, whole), NetOut: roundedBars(m.NetOut, whole),
	})
}

func (p Process) MarshalJSON() ([]byte, error) {
	// value: a share in a list by CPU, bytes in a list by memory, which
	// are whole: both to 2 decimals.
	type plain Process
	p.Value = rounded(p.Value, share)
	return json.Marshal(plain(p))
}

type usageJSON struct {
	CPU     float64   `json:"cpu"`
	Memory  int64     `json:"memory"`
	Started time.Time `json:"started,omitzero"`
}

func (u Usage) MarshalJSON() ([]byte, error) {
	return json.Marshal(usageJSON{CPU: rounded(u.CPU, share), Memory: u.Memory, Started: u.Started})
}

func (v VMUsage) MarshalJSON() ([]byte, error) {
	// invariant: written whole, its VM beside its usage: the Usage it
	// embeds would write itself alone.
	return json.Marshal(struct {
		VM string `json:"vm"`
		usageJSON
	}{VM: v.VM, usageJSON: usageJSON{CPU: rounded(v.CPU, share), Memory: v.Memory, Started: v.Started}})
}

func (s Sample) MarshalJSON() ([]byte, error) {
	type plain Sample
	for i := range s.Load {
		s.Load[i] = rounded(s.Load[i], share)
	}
	return json.Marshal(plain(s))
}
