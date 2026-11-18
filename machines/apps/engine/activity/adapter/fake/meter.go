package fake

import (
	"context"
	"errors"

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
		out.Sample = r.Answers(m.w.Now)
		return out, true
	}
	return world.Meter{Sample: world.Meter{}.Answers(m.w.Now)}, false
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
	return r.Sample, nil
}

func (m *Meter) Top(ctx context.Context) (activity.Busiest, error) {
	if err := ctx.Err(); err != nil {
		return activity.Busiest{}, err
	}
	r, _ := m.read()
	return activity.Busiest{CPU: activity.Ranked(r.Top.CPU), Memory: activity.Ranked(r.Top.Memory)}, nil
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
