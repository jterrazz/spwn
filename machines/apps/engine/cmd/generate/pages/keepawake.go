package pages

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	pagekeepawake "github.com/jterrazz/studio-station/apps/engine/pages/keepawake"
)

var keepAwakeNow = time.Date(2026, 11, 3, 11, 30, 0, 0, time.UTC)

func thisMacReading() awake.Reading {
	return awake.Reading{Lid: "open", Source: "ac", HasBattery: true, Percent: 80}
}

func keepAwakeReport(engine string, s *awake.Session, r awake.Reading) awake.Report {
	return awake.Report{Now: keepAwakeNow, Engine: engine, Reading: r, Session: s, Guards: awake.GuardStates(r, s)}
}

func heldSession() *awake.Session {
	start := keepAwakeNow.Add(-48 * time.Minute)
	return &awake.Session{Lid: true, StartedAt: start, Until: start.Add(4 * time.Hour), By: "station awake 4h --lid"}
}

func keepAwakePages() []Fixture {
	low := thisMacReading()
	low.Source, low.Percent = "battery", 28
	held := heldSession()
	out := states("keep-awake",
		pagekeepawake.Page(keepAwakeReport(awake.EnginePending, nil, thisMacReading()), thisMacContext()),
		pagekeepawake.Page(keepAwakeReport(awake.EngineRunning, nil, thisMacReading()), thisMacContext()),
		pagekeepawake.Page(keepAwakeReport(awake.EngineStopped, nil, thisMacReading()), thisMacContext()),
		pagekeepawake.Page(keepAwakeReport(awake.EngineRunning, held, low), thisMacContext()),
	)
	awoken := awake.Awoken{Session: *held}
	return append(out, states("outcome-keep-awake",
		pending(pagekeepawake.Awake(awoken, nil, thisMacContext())),
		pagekeepawake.Slept(awake.Slept{}, thisMacContext()),
		failure("outcome", "awake", "Awake stays off: the battery is at 18 % on battery", "station awake"),
		pagekeepawake.Awake(awoken, nil, thisMacContext()),
	)...)
}
