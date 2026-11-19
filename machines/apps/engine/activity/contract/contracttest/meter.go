package contracttest

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/activity/contract"
)

type Totals struct {
	// value: what the machine's whole physical disks and real links counted
	// since boot, and its memory pressure: the counters a sample must show.
	DiskRead, DiskWritten, NetIn, NetOut uint64
	Pressure                             activity.Pressure
}

type MeterSubject struct {
	Meter contract.Meter
	// value: the counters the subject's machine holds on its disks and links
	// that count, while its disk images, partitions and virtual links count too.
	Want Totals
	// value: a pid of a process the machine runs, and when it started.
	Running int
	Started time.Time
	// value: a meter of the same machine whose kernel answers nothing.
	Silent contract.Meter
}

func Meter(t *testing.T, subject func(t *testing.T) MeterSubject) {
	ctx := context.Background()
	busiestFirst := func(ps []activity.Process) bool {
		return len(ps) <= activity.Top && slices.IsSortedFunc(ps, func(a, b activity.Process) int {
			switch {
			case a.Value > b.Value:
				return -1
			case a.Value < b.Value:
				return 1
			}
			return 0
		})
	}

	t.Run("a sample counts whole physical disks and real links only", func(t *testing.T) {
		s := subject(t)
		got, err := s.Meter.Sample(ctx)
		if err != nil {
			t.Fatalf("Sample: %v", err)
		}
		totals := Totals{DiskRead: got.DiskRead, DiskWritten: got.DiskWritten, NetIn: got.NetIn, NetOut: got.NetOut, Pressure: got.Pressure}
		if totals != s.Want {
			t.Errorf("Sample() counted %+v; want %+v", totals, s.Want)
		}
		if got.At.IsZero() || got.CPUTotal == 0 || got.CPUBusy > got.CPUTotal || got.CPUs == 0 || got.MemTotal == 0 || got.MemUsed > got.MemTotal || got.Boot.IsZero() {
			t.Errorf("Sample() = %+v; want its time, CPU ticks, memory and boot", got)
		}
	})

	t.Run("a sample leaves thermal, battery and free space to their owners", func(t *testing.T) {
		got, _ := subject(t).Meter.Sample(ctx)
		if got.Thermal != activity.ThermalUnknown || got.Battery != nil || got.DiskFree != 0 || got.DiskTotal != 0 {
			t.Errorf("Sample() = %+v; want thermal unknown, no battery, no free space", got)
		}
	})

	t.Run("the busiest processes come first, at most the top", func(t *testing.T) {
		got, err := subject(t).Meter.Top(ctx)
		if err != nil {
			t.Fatalf("Top: %v", err)
		}
		if len(got.CPU) == 0 || len(got.Memory) == 0 || !busiestFirst(got.CPU) || !busiestFirst(got.Memory) {
			t.Errorf("Top() = %+v; want at most %d each, busiest first", got, activity.Top)
		}
	})

	t.Run("one process is read by its pid, and a gone one is an error", func(t *testing.T) {
		s := subject(t)
		if got, err := s.Meter.Process(ctx, s.Running); err != nil || got.Memory <= 0 || !got.Started.Equal(s.Started) {
			t.Errorf("Process(%d) = %+v, %v; want its resident memory, started at %v", s.Running, got, err, s.Started)
		}
		if _, err := s.Meter.Process(ctx, 999_999); err == nil {
			t.Error("Process() of a pid that runs nothing answered")
		}
	})

	t.Run("a machine whose kernel answers nothing reads nothing, never an error", func(t *testing.T) {
		s := subject(t)
		got, err := s.Silent.Sample(ctx)
		if err != nil || got.CPUTotal != 0 || got.MemTotal != 0 || !got.Boot.IsZero() {
			t.Errorf("Sample() = %+v, %v; want nothing read and no error", got, err)
		}
		top, err := s.Silent.Top(ctx)
		if err != nil || top.CPU == nil || top.Memory == nil || len(top.CPU)+len(top.Memory) != 0 {
			t.Errorf("Top() = %+v, %v; want empty lists and no error", top, err)
		}
	})

	t.Run("a read past its context is an error", func(t *testing.T) {
		s := subject(t)
		done, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := s.Meter.Sample(done); err == nil {
			t.Error("Sample() on an ended context answered")
		}
		if _, err := s.Meter.Top(done); err == nil {
			t.Error("Top() on an ended context answered")
		}
	})
}
