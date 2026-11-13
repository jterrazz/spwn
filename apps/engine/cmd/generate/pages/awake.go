package pages

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	pageawake "github.com/jterrazz/studio-station/apps/engine/pages/awake"
)

// value: Awake on every Mac: This Mac a laptop kept awake with its lid
// closed, the mini a desktop that never sleeps and hosts infra.
func awakeAllMachines(engine string, held bool) pageawake.AllMachines {
	laptop := awake.Report{Now: keepAwakeNow, Engine: engine, Mac: "jterrazz-macbook-pro", ThisMac: true, Reading: thisMacReading(),
		Caps: machine.Capabilities{Battery: true, Lid: true, Thermal: true, Engine: true}}
	mini := awake.Report{Now: keepAwakeNow, Engine: engine, Mac: "mac-mini", Reading: awake.Reading{Source: "ac"},
		Caps: machine.Capabilities{Thermal: true, Engine: true}, Servers: []string{"infra"}}
	if held {
		laptop.Session = heldSession()
		mini.NeverSleeps, mini.NeverHeld, mini.NeverSince = true, true, keepAwakeNow.Add(-12*24*time.Hour)
	}
	return pageawake.AllMachines{Now: keepAwakeNow, Reports: []awake.Report{laptop, mini}}
}

func awakePages() []Fixture {
	return states("awake",
		pageawake.Page(awakeAllMachines(awake.EngineReading, false), thisMacContext()),
		pageawake.Page(awakeAllMachines(awake.EngineRunning, false), thisMacContext()),
		failure("awake", "awake", "the engine is stopped on jterrazz-macbook-pro", "station engine start"),
		pageawake.Page(awakeAllMachines(awake.EngineRunning, true), thisMacContext()),
	)
}
