package activity

import (
	"math"
	"time"
)

type Bump struct {
	// value: a hill of load: how long ago its peak was, how wide it is, how
	// high it rises over the level.
	Ago    time.Duration `json:"ago"`
	Width  time.Duration `json:"width"`
	Height float64       `json:"height"`
}

type Profile struct {
	// value: how a machine's CPU and network went over a stretch of time: a
	// level, its slow swing, the bumps it had, an optional plateau between
	// two absolute times, and the network's levels in MB/s. Every moment but
	// the plateau is told relative to the `now` its points are asked at.
	Base    float64       `json:"base"`
	Swing   float64       `json:"swing"`
	Seed    float64       `json:"seed"`
	Bumps   []Bump        `json:"bumps,omitempty"`
	Plateau *[2]time.Time `json:"plateau,omitempty"`
	In      float64       `json:"in"`
	Out     float64       `json:"out"`
}

const mb = 1e6

func (p Profile) Points(now time.Time, span, every time.Duration) []Sample {
	// value: the points a profile gives every `every` over `span` up to now,
	// oldest first, its counters accumulating each interval's share and rates.
	n := int(span / every)
	out := make([]Sample, 0, n+1)
	s := Sample{At: now.Add(-span), CPUTotal: 1_000_000}
	out = append(out, s)
	for i := 1; i <= n; i++ {
		at := now.Add(-span + time.Duration(i)*every)
		seconds := every.Seconds()
		in, outRate := p.rates(at, now)
		s = Sample{At: at, CPUTotal: s.CPUTotal + uint64(seconds*1000), CPUBusy: s.CPUBusy + uint64(p.cpu(at, now)*seconds*10),
			NetIn: s.NetIn + uint64(in*seconds), NetOut: s.NetOut + uint64(outRate*seconds)}
		out = append(out, s)
	}
	return out
}

func (p Profile) cpu(t, now time.Time) float64 {
	if p.Plateau != nil && !t.Before(p.Plateau[0]) && !t.After(p.Plateau[1]) {
		if t.Equal(p.Plateau[0].Add(12 * time.Minute)) {
			return 94.4
		}
		return 94
	}
	m := t.Sub(now).Minutes()
	sec := t.Sub(now).Seconds()
	v := p.Base + p.Swing*math.Sin(m/7+p.Seed) + p.Swing/2*math.Sin(m/2.3+2*p.Seed) + p.Swing/2*math.Sin(sec/23+p.Seed) + p.Swing/3*math.Sin(sec/9+2*p.Seed)
	for _, b := range p.Bumps {
		d := t.Sub(now.Add(-b.Ago)).Minutes() / b.Width.Minutes()
		v += b.Height * math.Exp(-d*d)
	}
	return min(max(v, 1), 99)
}

// value: the network's rates in bytes per second: a steady level and a
// burst received every 7 min, sent every 5.
func (p Profile) rates(t, now time.Time) (in, out float64) {
	m := t.Sub(now).Minutes()
	burst := func(every float64) float64 {
		d := math.Mod(math.Abs(m)+every, every) - every/2
		return math.Exp(-d * d * 4)
	}
	sway := func(seed float64) float64 { return 1 + 0.2*math.Sin(m/3.7+seed) + 0.12*math.Sin(m*1.3+2*seed) }
	return p.In * (sway(p.Seed) + 1.2*burst(7)) * mb, p.Out * (sway(p.Seed+1) + 0.8*burst(5)) * mb
}
