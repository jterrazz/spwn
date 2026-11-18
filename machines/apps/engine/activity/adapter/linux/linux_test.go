package linux

import (
	"testing"
	"testing/fstest"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/activity/contract/contracttest"
)

type clock struct{}

func (clock) Now() time.Time { return time.Unix(1_800_000_000, 0) }

func file(text string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(text)} }

func TestMeter(t *testing.T) {
	vm := fstest.MapFS{
		"proc/stat":            file("cpu  100 0 50 800 50 0 0 0 0 0\ncpu0 50 0 25 400 25 0 0 0 0 0\ncpu1 50 0 25 400 25 0 0 0 0 0\n"),
		"proc/loadavg":         file("1.40 1.50 1.60 2/300 4242\n"),
		"proc/uptime":          file("3600.50 7000.00\n"),
		"proc/meminfo":         file("MemTotal:        4028096 kB\nMemFree:  1000 kB\nMemAvailable: 2014048 kB\nSwapTotal: 0 kB\nSwapFree: 0 kB\n"),
		"proc/pressure/memory": file("some avg10=25.00 avg60=1.00 avg300=0.50 total=100\nfull avg10=0.00 avg60=0.00 avg300=0.00 total=0\n"),
		"proc/diskstats":       file(" 253 0 vda 10 0 2000 0 20 0 4000 0 0 0 0\n 253 1 vda1 10 0 2000 0 20 0 4000 0 0 0 0\n 7 0 loop0 5 0 900 0 0 0 0 0 0 0 0\n"),
		"proc/net/dev":         file("Inter-|   Receive\n face |bytes\n    lo: 500 5 0 0 0 0 0 0 500 5 0 0 0 0 0 0\n  eth0: 9000 90 0 0 0 0 0 0 4000 40 0 0 0 0 0 0\n veth1a2b: 700 7 0 0 0 0 0 0 700 7 0 0 0 0 0 0\n"),
		"proc/1/stat":          file("1 (systemd) S 0 1 1 0 -1 0 0 0 0 0 10 5 0 0 20 0 1 0 1 0 0\n"),
		"proc/1/statm":         file("4000 2250 100 1 0 300 0\n"),
		"proc/42/stat":         file("42 (node) S 1 42 42 0 -1 0 0 0 0 0 200 50 0 0 20 0 1 0 100 0 0\n"),
		"proc/42/statm":        file("90000 51200 100 1 0 300 0\n"),
	}
	contracttest.Meter(t, func(*testing.T) contracttest.MeterSubject {
		return contracttest.MeterSubject{
			Meter: New(vm, clock{}), Want: contracttest.Totals{DiskRead: 2000 * 512, DiskWritten: 4000 * 512, NetIn: 9000, NetOut: 4000, Pressure: activity.PressureWarn},
			Running: 42, Started: time.Unix(1_800_000_000-3600, 0), Silent: New(fstest.MapFS{}, clock{}),
		}
	})
	s, _ := New(vm, clock{}).Sample(t.Context())
	if s.MemUsed != 2014048*1024 || s.CPUBusy != 150 || s.CPUTotal != 1000 || s.CPUs != 2 {
		t.Errorf("Sample() = %+v; want total minus available, the busy and all ticks, 2 CPUs", s)
	}
}
