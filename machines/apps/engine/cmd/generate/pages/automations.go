package pages

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/automations"
	pageautomations "github.com/jterrazz/studio-station/apps/engine/pages/automations"
)

// value: the automations the project declares on This Mac: one cron with a
// run that failed, one file watch that never ran, one named like the verb.
func automationList() []automations.Automation {
	return []automations.Automation{
		{Name: "inbox", Agent: "trinity", Trigger: automations.Trigger{Kind: automations.FS, Spec: "./inbox"}},
		{Name: "log", Agent: "neo", Trigger: automations.Trigger{Kind: automations.Cron, Spec: "0 9 * * 1"}},
		{Name: "morning", Agent: "neo", Trigger: automations.Trigger{Kind: automations.Cron, Spec: "0 6 * * *"},
			LastFired: time.Date(2026, time.May, 2, 6, 0, 0, 0, time.UTC), Fires: 2, Failed: 1},
	}
}

// value: the runs written for the project: one that passed, one that failed.
func automationRuns() []automations.Run {
	return []automations.Run{
		{Name: "morning", Trigger: automations.Cron, Cause: "catchup", Scheduled: time.Date(2026, time.May, 1, 6, 0, 0, 0, time.UTC), At: time.Date(2026, time.May, 1, 6, 0, 0, 0, time.UTC), DurationMS: 1500, OK: true, Output: "brief written", Missed: 2},
		{Name: "morning", Trigger: automations.Cron, Cause: "on-time", Scheduled: time.Date(2026, time.May, 2, 6, 0, 0, 0, time.UTC), At: time.Date(2026, time.May, 2, 6, 0, 0, 0, time.UTC), DurationMS: 40, Error: "no running world"},
	}
}

func automationPages() []Fixture {
	return append(states("automations",
		pending(pageautomations.List(automationList(), thisMacContext())),
		pageautomations.List(nil, thisMacContext()),
		failure("automations", "automations", "the manifest of this project cannot be read", "station automations"),
		pageautomations.List(automationList(), thisMacContext()),
	), states("automations-log",
		pending(pageautomations.Log(automations.Runs{Runs: automationRuns(), Total: 3, Unreadable: 1}, thisMacContext())),
		pageautomations.Log(automations.Runs{}, thisMacContext()),
		failure("automations-log", "automations log", "the runs of this project cannot be read", "station automations log"),
		pageautomations.Log(automations.Runs{Runs: automationRuns(), Total: 3, Unreadable: 1}, thisMacContext()),
	)...)
}
