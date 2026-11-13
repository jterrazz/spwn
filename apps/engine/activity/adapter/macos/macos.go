package macos

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/activity/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/darwin"
)

const (
	// value: the shortest time between two process tables a CPU share is
	// measured over, and the oldest table a share is still measured from.
	window = 250 * time.Millisecond
	stale  = time.Minute
)

var ErrNoProcess = errors.New("no such process")

type Kernel interface {
	Counters() (darwin.Counters, error)
	Processes() (darwin.Procs, error)
	Process(pid int) (darwin.Procs, error)
}

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

type Meter struct {
	kernel Kernel
	clock  Clock
	mu     sync.Mutex
	// value: the last process table read, the start of the next share.
	last darwin.Procs
	cpus int
}

var _ contract.Meter = (*Meter)(nil)

func New(k Kernel, clock Clock) *Meter { return &Meter{kernel: k, clock: clock} }

func (m *Meter) Sample(ctx context.Context) (activity.Sample, error) {
	if err := ctx.Err(); err != nil {
		return activity.Sample{}, err
	}
	now := m.clock.Now().UTC()
	s := activity.Sample{At: now, Thermal: activity.ThermalUnknown}
	if c, err := m.kernel.Counters(); err == nil {
		m.fill(&s, c)
	}
	return s, nil
}

func (m *Meter) fill(s *activity.Sample, c darwin.Counters) {
	for _, t := range c.CPUs {
		busy := t.User + t.System + t.Nice
		s.CPUBusy += busy
		s.CPUTotal += busy + t.Idle
	}
	s.CPUs, s.Load = c.Cores(), c.Load
	m.mu.Lock()
	m.cpus = s.CPUs
	m.mu.Unlock()
	p := c.Pages
	s.MemUsed = bytes((p.Internal - min(p.Purgeable, p.Internal) + p.Wired + p.Compressor) * p.Size)
	s.MemTotal, s.SwapUsed, s.Pressure = bytes(c.Memsize), bytes(c.SwapUsed), pressure(c.Pressure)
	for _, d := range c.Disks {
		if physical(d) {
			s.DiskRead += d.Read
			s.DiskWritten += d.Written
		}
	}
	for _, l := range c.Links {
		if realLink(l.Name) {
			s.NetIn += l.In
			s.NetOut += l.Out
		}
	}
	if c.Up > 0 {
		s.Boot = s.At.Add(-c.Up).Truncate(time.Second)
	}
}

func bytes(n uint64) int64 { return int64(min(n, 1<<63-1)) } //nolint:gosec // outside: clamped to an int64 first.

// outside: kern.memorystatus_vm_pressure_level says 1 normal, 2 warn, 4 critical.
func pressure(level int) activity.Pressure {
	switch level {
	case 1:
		return activity.PressureNormal
	case 2:
		return activity.PressureWarn
	case 4:
		return activity.PressureCritical
	}
	return ""
}

// outside: IOKit says where a block storage device sits: the internal SSD is
// Internal, a disk image a Virtual Interface in a File, so its bytes count once,
// on the disk that holds the image.
func physical(d darwin.BlockDriver) bool {
	return d.Location == "Internal" && d.Interconnect != "Virtual Interface"
}

// outside: Ethernet and Wi-Fi are en*; lo0, utun* (Tailscale, VPNs), bridge*
// (VM traffic), awdl*, llw*, ap* and anpi* ride or loop over them, so their
// bytes are counted already.
func realLink(name string) bool { return strings.HasPrefix(name, "en") }

func (m *Meter) Top(ctx context.Context) (activity.Busiest, error) {
	empty := activity.Busiest{CPU: []activity.Process{}, Memory: []activity.Process{}}
	before, after, err := m.measure(ctx, m.kernel.Processes)
	if err != nil {
		if ctx.Err() != nil {
			return activity.Busiest{}, ctx.Err()
		}
		return empty, nil
	}
	shares := m.shares(before, after)
	out := empty
	for _, p := range after.List {
		out.CPU = append(out.CPU, activity.Process{PID: p.PID, Name: p.Name, Value: shares[p.PID]})
		out.Memory = append(out.Memory, activity.Process{PID: p.PID, Name: p.Name, Value: float64(p.Resident)})
	}
	return activity.Busiest{CPU: busiest(out.CPU), Memory: busiest(out.Memory)}, nil
}

func (m *Meter) Process(ctx context.Context, pid int) (activity.Usage, error) {
	read := func() (darwin.Procs, error) { return m.kernel.Process(pid) }
	before, after, err := m.measure(ctx, read)
	if err != nil {
		if ctx.Err() != nil {
			return activity.Usage{}, ctx.Err()
		}
		return activity.Usage{}, ErrNoProcess
	}
	return activity.Usage{CPU: m.shares(before, after)[pid], Memory: bytes(after.List[0].Resident), Started: after.List[0].Started}, nil
}

func (m *Meter) measure(ctx context.Context, read func() (darwin.Procs, error)) (before, after darwin.Procs, err error) {
	// invariant: a CPU share is measured between two tables; the last one
	// read is the start when it is recent enough, else a second table is
	// read a window after the first.
	if ctxErr := ctx.Err(); ctxErr != nil {
		return before, after, ctxErr
	}
	after, err = read()
	if err != nil || len(after.List) == 0 {
		return before, after, cmp.Or(err, ErrNoProcess)
	}
	m.mu.Lock()
	before = m.last
	m.mu.Unlock()
	if gap := after.At - before.At; before.List == nil || gap < window || gap > stale {
		if slept := m.clock.Sleep(ctx, window); slept != nil {
			return before, after, slept
		}
		before = after
		if after, err = read(); err != nil || len(after.List) == 0 {
			return before, after, cmp.Or(err, ErrNoProcess)
		}
	}
	if len(after.List) > 1 {
		m.mu.Lock()
		m.last = after
		m.mu.Unlock()
	}
	return before, after, nil
}

func (m *Meter) shares(before, after darwin.Procs) map[int]float64 {
	// value: each process's CPU share of the whole machine between the two
	// tables, 0 to 100.
	m.mu.Lock()
	cpus := m.cpus
	m.mu.Unlock()
	if cpus == 0 {
		if c, err := m.kernel.Counters(); err == nil {
			cpus = c.Cores()
		}
	}
	cpus = max(cpus, 1)
	out := map[int]float64{}
	elapsed := after.At - before.At
	if elapsed <= 0 {
		return out
	}
	used := map[int]time.Duration{}
	for _, p := range before.List {
		used[p.PID] = p.CPU
	}
	for _, p := range after.List {
		if was, ok := used[p.PID]; ok && p.CPU > was {
			out[p.PID] = min(100, 100*float64(p.CPU-was)/float64(elapsed)/float64(cpus))
		}
	}
	return out
}

func busiest(ps []activity.Process) []activity.Process {
	slices.SortStableFunc(ps, func(a, b activity.Process) int {
		return cmp.Or(cmp.Compare(b.Value, a.Value), cmp.Compare(a.PID, b.PID))
	})
	return ps[:min(len(ps), activity.Top)]
}
