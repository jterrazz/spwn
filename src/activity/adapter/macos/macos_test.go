package macos

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/activity"
	"github.com/jterrazz/studio-station/src/activity/contract/contracttest"
	"github.com/jterrazz/studio-station/src/platform/darwin"
)

type kernel struct {
	counters darwin.Counters
	tables   []darwin.Procs
	started  map[int]time.Time
	read     int
}

func (k *kernel) Counters() (darwin.Counters, error) {
	if k.counters.CPUs == nil {
		return darwin.Counters{}, errors.New("silent")
	}
	return k.counters, nil
}

func (k *kernel) Processes() (darwin.Procs, error) {
	if len(k.tables) == 0 {
		return darwin.Procs{}, errors.New("silent")
	}
	t := k.tables[min(k.read, len(k.tables)-1)]
	k.read++
	return t, nil
}

func (k *kernel) Process(pid int) (darwin.Procs, error) {
	t, err := k.Processes()
	for _, p := range t.List {
		if p.PID == pid {
			p.Started = k.started[pid]
			return darwin.Procs{At: t.At, List: []darwin.Proc{p}}, err
		}
	}
	return darwin.Procs{}, errors.New("gone")
}

type clock struct{}

func (clock) Now() time.Time { return time.Unix(1_800_000_000, 0) }

func (clock) Sleep(ctx context.Context, _ time.Duration) error { return ctx.Err() }

func mac() *kernel {
	return &kernel{
		counters: darwin.Counters{
			CPUs:    []darwin.Ticks{{User: 30, System: 10, Idle: 60}, {User: 10, System: 10, Idle: 80}},
			Load:    [3]float64{2.8, 3.1, 2.9},
			Pages:   darwin.Pages{Size: 16384, Internal: 400_000, Purgeable: 10_000, Wired: 200_000, Compressor: 150_000},
			Memsize: 16 << 30, SwapUsed: 1 << 30, Pressure: 4, Up: 76 * time.Hour,
			Disks: []darwin.BlockDriver{
				{Media: "disk0", Interconnect: "Apple Fabric", Location: "Internal", Read: 182e9, Written: 96e9},
				{Media: "disk24", Interconnect: "Virtual Interface", Location: "File", Read: 125e9},
			},
			Links: []darwin.Link{{Name: "lo0", In: 7e8, Out: 7e8}, {Name: "en0", In: 38e9, Out: 4.6e9}, {Name: "utun4", In: 1e9, Out: 1e9}, {Name: "bridge100", In: 2e9, Out: 2e9}},
		},
		started: map[int]time.Time{302: time.Unix(1_799_000_000, 0)},
		tables: []darwin.Procs{
			{At: time.Second, List: []darwin.Proc{{PID: 301, Name: "Zed", CPU: time.Second, Resident: 1.9e9}, {PID: 302, Name: "node", CPU: 2 * time.Second, Resident: 4.1e9}}},
			{At: 2 * time.Second, List: []darwin.Proc{{PID: 301, Name: "Zed", CPU: 1100 * time.Millisecond, Resident: 1.9e9}, {PID: 302, Name: "node", CPU: 2360 * time.Millisecond, Resident: 4.1e9}}},
		},
	}
}

func TestMeter(t *testing.T) {
	contracttest.Meter(t, func(*testing.T) contracttest.MeterSubject {
		return contracttest.MeterSubject{
			Meter:   New(mac(), clock{}),
			Want:    contracttest.Totals{DiskRead: 182e9, DiskWritten: 96e9, NetIn: 38e9, NetOut: 4.6e9, Pressure: activity.PressureCritical},
			Running: 302,
			Started: time.Unix(1_799_000_000, 0),
			Silent:  New(&kernel{}, clock{}),
		}
	})
}

func TestMemoryUsedIsAppWiredAndCompressed(t *testing.T) {
	s, _ := New(mac(), clock{}).Sample(context.Background())
	if want := int64((400_000 - 10_000 + 200_000 + 150_000) * 16384); s.MemUsed != want {
		t.Errorf("MemUsed = %d; want app memory, wired and compressed, %d", s.MemUsed, want)
	}
}

func TestAProcessShareIsOfTheWholeMachine(t *testing.T) {
	top, _ := New(mac(), clock{}).Top(context.Background())
	if top.CPU[0].Name != "node" || top.CPU[0].Value != 18 || top.CPU[1].Value != 5 {
		t.Errorf("Top().CPU = %+v; want node 18 %% then Zed 5 %% of the 2 CPUs", top.CPU)
	}
}

func TestACountWithoutTicksIsStillTheCPUsAndTheMemory(t *testing.T) {
	// value: a build without cgo reads the count and the size, no per-CPU tick.
	k := &kernel{counters: darwin.Counters{CPUs: []darwin.Ticks{}, CPUCount: 8, Memsize: 16 << 30}}
	s, _ := New(k, clock{}).Sample(context.Background())
	if s.CPUs != 8 || s.MemTotal != 16<<30 {
		t.Errorf("cpus %d, memory %d", s.CPUs, s.MemTotal)
	}
}
