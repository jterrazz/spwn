package pages

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/automations"
	pageautomations "github.com/jterrazz/studio-station/apps/engine/pages/automations"
)

// value: the automations the project declares on This Mac: one cron with a
// run that failed, one file watch that never ran.
func automationList() []automations.Automation {
	return []automations.Automation{
		{Name: "inbox", Agent: "trinity", Trigger: automations.Trigger{Kind: automations.FS, Spec: "./inbox"}},
		{Name: "morning", Agent: "neo", Trigger: automations.Trigger{Kind: automations.Cron, Spec: "0 6 * * *"},
			LastFired: time.Date(2026, time.May, 2, 6, 0, 0, 0, time.UTC), Fires: 2, Failed: 1},
	}
}

func automationPages() []Fixture {
	return states("automations",
		pending(pageautomations.List(automationList(), thisMacContext())),
		pageautomations.List(nil, thisMacContext()),
		failure("automations", "automations", "the manifest of this project cannot be read", "station automations"),
		pageautomations.List(automationList(), thisMacContext()),
	)
}
