package fake

import (
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/activity/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestMeter(t *testing.T) {
	contracttest.Meter(t, func(*testing.T) contracttest.MeterSubject {
		busy := world.Meter{
			Sample: activity.Sample{
				At: time.Unix(1_800_000_000, 0), CPUBusy: 40, CPUTotal: 100, CPUs: 10, MemUsed: 12e9, MemTotal: 16 << 30, Pressure: activity.PressureNormal,
				DiskRead: 182e9, DiskWritten: 96e9, NetIn: 38e9, NetOut: 4.6e9, Boot: time.Unix(1_799_700_000, 0),
				Thermal: 0, Battery: &activity.Battery{Percent: 96}, DiskFree: 388e9, DiskTotal: 994e9,
			},
			Top: activity.Busiest{
				CPU:    []activity.Process{{PID: 2, Name: "node", Value: 12}, {PID: 1, Name: "station", Value: 40}},
				Memory: []activity.Process{{PID: 2, Name: "node", Value: 2e9}},
			},
			Processes: map[int]activity.Usage{2: {CPU: 12, Memory: 2e9, Started: time.Unix(1_799_000_000, 0)}},
		}
		return contracttest.MeterSubject{
			Meter:   NewMeter(busy),
			Want:    contracttest.Totals{DiskRead: 182e9, DiskWritten: 96e9, NetIn: 38e9, NetOut: 4.6e9, Pressure: activity.PressureNormal},
			Running: 2,
			Started: time.Unix(1_799_000_000, 0),
			Silent:  NewMeter(world.Meter{}),
		}
	})
}
