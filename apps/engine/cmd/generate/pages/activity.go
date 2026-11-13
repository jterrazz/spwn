package pages

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	pageactivity "github.com/jterrazz/studio-station/apps/engine/pages/activity"
)

// value: This Mac's activity as the engine keeps it: 34 % busy of 10 CPU,
// rates over the last 10 s, the VM linux-1 the busiest.
func thisMacActivity() pageactivity.View {
	at := keepAwakeNow
	last := activity.Sample{
		At: at, CPUs: 10, Load: [3]float64{2.8, 3.1, 2.9}, MemUsed: 13314398618, MemTotal: 17179869184, Pressure: activity.PressureNormal,
		SwapUsed: 1288490188, DiskRead: 182e9, DiskWritten: 96e9, NetIn: 38e9, NetOut: 4.6e9, DiskFree: 388e9, DiskTotal: 994e9,
		Thermal: 0, Battery: &activity.Battery{Percent: 96, Charging: true, Health: 98, Cycles: 112}, Boot: at.Add(-76 * time.Hour),
	}
	top := activity.Busiest{
		CPU:    []activity.Process{{PID: 1001, Name: "Virtualization", Value: 18, VM: "linux-1"}, {PID: 4100, Name: "Zed", Value: 9}, {PID: 310, Name: "WindowServer", Value: 4}},
		Memory: []activity.Process{{PID: 1001, Name: "Virtualization", Value: 4402341478, VM: "linux-1"}, {PID: 4100, Name: "Zed", Value: 2040109465}},
	}
	r := activity.Report{Source: activity.FromEngine, Since: at.Add(-3 * time.Hour), Last: activity.Reading{Sample: last, Top: top}, Window: activity.RateWindow,
		Rates: activity.Rates{CPU: 34, DiskRead: 4.1e6, DiskWrite: 1.3e6, NetIn: 2.1e6, NetOut: 210e3}}
	return pageactivity.View{Name: "jterrazz-macbook-pro", Label: "This Mac", Kind: "computer", ThisMac: true, Report: r}
}

func activityPages() []Fixture {
	stopped := pageactivity.View{Name: "api", Label: "api", Kind: "workspace", Mac: "jterrazz-macbook-pro", On: "This Mac", Stopped: true}
	return states("activity",
		pending(pageactivity.Page(thisMacActivity(), thisMacContext())),
		pageactivity.Page(stopped, thisMacContext()),
		failure("activity", "activity", "jterrazz-mac-mini does not answer", "station machines"),
		pageactivity.Page(thisMacActivity(), thisMacContext()),
	)
}
