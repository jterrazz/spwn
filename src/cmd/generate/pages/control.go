package pages

import (
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/engine"
	pageengine "github.com/jterrazz/studio-station/src/pages/engine"
)

func controlPages() []Fixture {
	stopped := engine.StopReport{Engine: engine.EngineHalted, Awake: engine.AwakeEnded, Sharing: true, VMs: []engine.Stopped{
		{Name: "linux-1", Lifetime: "ephemeral", Result: engine.ResultStopped},
		{Name: "api", Lifetime: "persistent", Result: engine.ResultStopped},
	}}
	stuck := engine.StopReport{Engine: engine.EngineKept, VMs: []engine.Stopped{
		{Name: "linux-1", Lifetime: "ephemeral", Result: engine.ResultStopped},
		{Name: "api", Lifetime: "persistent", Result: engine.ResultFailed, Detail: catalog.HypervisorApp + " stop exited 1"},
	}}
	started := engine.StartReport{Version: "v3.1.0", PID: 812}
	until := time.Date(2026, 11, 3, 18, 0, 0, 0, time.UTC)
	running := engine.Reading{State: engine.EngineRunning, Version: "v3.1.0", Station: "v3.1.0", Up: 74 * time.Hour, VMs: 2, AwakeUntil: &until}
	halted := engine.Reading{State: engine.EngineStopped, Station: "v3.1.0", Stopped: engine.StopMark{At: time.Date(2026, 11, 3, 13, 40, 0, 0, time.UTC), By: "station engine stop"}}
	out := states("engine",
		pending(pageengine.Read(running, time.UTC, thisMacContext())),
		pageengine.Read(halted, time.UTC, thisMacContext()),
		failure("engine", "engine", "the engine runs v3.0.0, station is v3.1.0", "station engine start"),
		pageengine.Read(running, time.UTC, thisMacContext()),
	)
	out = append(out, states("outcome-stop",
		pending(pageengine.Stop(stopped, thisMacContext())),
		pageengine.Stop(engine.StopReport{Engine: engine.EngineAlready}, thisMacContext()),
		pageengine.Stop(stuck, thisMacContext()),
		pageengine.Stop(stopped, thisMacContext()),
	)...)
	return append(out, states("outcome-start",
		pending(pageengine.Start(started, thisMacContext())),
		pageengine.Start(engine.StartReport{Already: true, Version: "v3.1.0", PID: 812}, thisMacContext()),
		failure("outcome", "engine start", "the engine did not start: launchctl bootstrap exited 5", "station engine"),
		pageengine.Start(started, thisMacContext()),
	)...)
}
