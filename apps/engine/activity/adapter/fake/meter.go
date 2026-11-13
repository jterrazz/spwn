package fake

import (
	"cmp"
	"context"
	"errors"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

var (
	ErrNoProcess = errors.New("no such process")
	// value: a Meter the world scripted silent (world fault Meter.Sample).
	ErrSilent = errors.New("the meter does not answer")
)

type Meter struct {
	// invariant: the counters of one Mac of the world.
	w   *world.World
	mac string
}

func MeterOf(w *world.World, mac string) *Meter { return &Meter{w: w, mac: mac} }

func NewMeter(r world.Meter) *Meter {
	// value: a machine whose kernel answers r, at r's time; the zero Meter
	// answers nothing.
	w := &world.World{ThisMac: "this-mac", Now: r.Sample.At}
	w.Mac("this-mac").Meter = &r
	return MeterOf(w, "this-mac")
}

func (m *Meter) read() (world.Meter, bool) {
	// invariant: the counters are the world's, read at the world's now.
	m.w.Lock()
	defer m.w.Unlock()
	if r := m.w.Mac(m.mac).Meter; r != nil {
		out := *r
		if r.PerSecond != nil && !r.Sample.At.IsZero() {
			out.Sample = grown(r.Sample, *r.PerSecond, m.w.Now.Sub(r.Sample.At).Seconds())
		}
		out.Sample.At = m.w.Now
		return out, true
	}
	return world.Meter{Sample: activity.Sample{At: m.w.Now}}, false
}

func (m *Meter) Sample(ctx context.Context) (activity.Sample, error) {
	if err := ctx.Err(); err != nil {
		return activity.Sample{}, err
	}
	m.w.Lock()
	silent := m.w.Fails("Meter.Sample")
	m.w.Unlock()
	if silent {
		return activity.Sample{}, ErrSilent
	}
	r, _ := m.read()
	s := r.Sample
	s.Thermal, s.Battery, s.DiskFree, s.DiskTotal = activity.ThermalUnknown, nil, 0, 0
	return s, nil
}

func (m *Meter) Top(ctx context.Context) (activity.Busiest, error) {
	if err := ctx.Err(); err != nil {
		return activity.Busiest{}, err
	}
	r, _ := m.read()
	return activity.Busiest{CPU: top(r.Top.CPU), Memory: top(r.Top.Memory)}, nil
}

func (m *Meter) Process(ctx context.Context, pid int) (activity.Usage, error) {
	if err := ctx.Err(); err != nil {
		return activity.Usage{}, err
	}
	r, _ := m.read()
	u, ok := r.Processes[pid]
	if !ok {
		return activity.Usage{}, ErrNoProcess
	}
	return u, nil
}

func top(ps []activity.Process) []activity.Process {
	out := slices.Clone(ps)
	if out == nil {
		out = []activity.Process{}
	}
	slices.SortStableFunc(out, func(a, b activity.Process) int { return cmp.Compare(b.Value, a.Value) })
	return out[:min(len(out), activity.Top)]
}

func grown(s, rate activity.Sample, seconds float64) activity.Sample {
	if seconds < 0 {
		seconds = 0
	}
	by := func(n, per uint64) uint64 { return n + uint64(float64(per)*seconds) }
	s.CPUBusy, s.CPUTotal = by(s.CPUBusy, rate.CPUBusy), by(s.CPUTotal, rate.CPUTotal)
	s.DiskRead, s.DiskWritten = by(s.DiskRead, rate.DiskRead), by(s.DiskWritten, rate.DiskWritten)
	s.NetIn, s.NetOut = by(s.NetIn, rate.NetIn), by(s.NetOut, rate.NetOut)
	s.SwapUsed += int64(float64(rate.SwapUsed) * seconds)
	return s
}
