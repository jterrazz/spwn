package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/activity"
)

type Meter interface {
	// invariant: the machine's counters, read without starting a process; a
	// counter the machine did not answer is zero, never an error. Thermal is
	// activity.ThermalUnknown, Battery nil and DiskFree zero: those are Power's
	// and the Disk family's readings, composed in by the sample job.
	Sample(ctx context.Context) (activity.Sample, error)
	// invariant: the busiest processes by CPU and by memory, at most
	// activity.Top each, busiest first; a list nothing answered is empty.
	Top(ctx context.Context) (activity.Busiest, error)
	// invariant: one process's usage; an error when no such process runs.
	Process(ctx context.Context, pid int) (activity.Usage, error)
}
