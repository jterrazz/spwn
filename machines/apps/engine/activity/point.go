package activity

import (
	"encoding/json"
	"time"
)

type Range struct {
	Mean float64 `json:"mean"`
	Max  float64 `json:"max"`
}

type Rated struct {
	// value: what a bucket's rates measured: the seconds they span, the
	// mean and the busiest of each (a CPU share 0 to 100, bytes a second)
	// and when its CPU was busiest.
	Span      float64   `json:"span"`
	CPU       Range     `json:"cpu"`
	CPUMaxAt  time.Time `json:"cpu_max_at"`
	DiskRead  Range     `json:"disk_read"`
	DiskWrite Range     `json:"disk_write"`
	NetIn     Range     `json:"net_in"`
	NetOut    Range     `json:"net_out"`
}

type Point struct {
	// invariant: one bucket of a tier as a reader draws it, at its newest
	// sample: the samples it counts, its rates (nil a gap: nothing in it was
	// measured against the sample before, after a pause or a counter that
	// went back), its memory and swap in bytes, the least pressure it said.
	// Its load, free disk bytes and battery are its newest sample's, kept by
	// T2 alone: zero on a point of any other tier.
	At       time.Time  `json:"at"`
	Samples  int        `json:"samples"`
	Rated    *Rated     `json:"rated,omitempty"`
	Memory   Range      `json:"memory"`
	Swap     Range      `json:"swap"`
	MemTotal int64      `json:"mem_total"`
	CPUs     int        `json:"cpus"`
	Pressure Pressure   `json:"pressure,omitempty"`
	Load     [3]float64 `json:"load,omitzero"`
	DiskFree int64      `json:"disk_free,omitempty"`
	Battery  *Battery   `json:"battery,omitempty"`
}

func (b *bucket) point(every int64, g gauges, kept bool, last *Sample) Point {
	n := b.n()
	p := Point{
		At: time.UnixMilli(b.at).UTC(), Samples: n,
		Memory: Range{float64(b.mem.mean(n)), float64(b.mem.max)}, Swap: Range{float64(b.swap.mean(n)), float64(b.swap.max)},
		MemTotal: last.MemTotal, CPUs: last.CPUs, Pressure: pressureOf(b.floor()),
	}
	if kept {
		p.Load, p.DiskFree, p.Battery = g.loads(), g.free, g.charge()
	}
	if b.span > 0 {
		r := func(e envelope) Range { return Range{float64(e.mean), float64(e.max)} }
		p.Rated = &Rated{
			Span: float64(b.span), CPU: r(b.cpu), CPUMaxAt: time.UnixMilli(b.peakAt(every)).UTC(),
			DiskRead: r(b.diskRead), DiskWrite: r(b.diskWrite), NetIn: r(b.netIn), NetOut: r(b.netOut),
		}
	}
	return p
}

func (r Rated) Rounded() Rated {
	// value: r as an answer writes it: a share and a span to 2 decimals,
	// bytes a second whole.
	in := func(x Range, u unit) Range { return Range{rounded(x.Mean, u), rounded(x.Max, u)} }
	return Rated{
		Span: rounded(r.Span, share), CPU: in(r.CPU, share), CPUMaxAt: r.CPUMaxAt,
		DiskRead: in(r.DiskRead, whole), DiskWrite: in(r.DiskWrite, whole), NetIn: in(r.NetIn, whole), NetOut: in(r.NetOut, whole),
	}
}

func (r Rated) MarshalJSON() ([]byte, error) {
	type plain Rated
	return json.Marshal(plain(r.Rounded()))
}

func (p Point) MarshalJSON() ([]byte, error) {
	type plain Point
	for i := range p.Load {
		p.Load[i] = rounded(p.Load[i], share)
	}
	return json.Marshal(plain(p))
}
