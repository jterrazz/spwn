package pages

import (
	"github.com/jterrazz/studio-station/apps/engine/background"
	pagebackground "github.com/jterrazz/studio-station/apps/engine/pages/background"
)

// value: the estate's Mac's background tasks, one in each group and state:
// a failing one, a root orphan whose app is gone, the engine, a known app's,
// an unknown one, and one the job manager did not answer for.
func backgroundJobs() []background.Job {
	agent := func(label string) string { return "/Users/jterrazz/Library/LaunchAgents/" + label + ".plist" }
	rootPlist := func(label string) string { return "/Library/LaunchDaemons/" + label + ".plist" }
	return []background.Job{
		{Label: "com.jterrazz.brain.x-snapshot", File: agent("com.jterrazz.brain.x-snapshot"), Scope: background.ScopeUser, Loaded: true, LastExit: 2, ProgramExists: true, OwnerExists: true},
		{Label: "com.bjango.istatmenus.installer", File: rootPlist("com.bjango.istatmenus.installer"), Scope: background.ScopeRoot, Loaded: true, ProgramExists: true,
			Removal: "sudo launchctl bootout system/com.bjango.istatmenus.installer && sudo rm /Library/LaunchDaemons/com.bjango.istatmenus.installer.plist"},
		{Label: "com.jterrazz.station.engine", File: agent("com.jterrazz.station.engine"), Scope: background.ScopeUser, Loaded: true, Running: true, PID: 812, ProgramExists: true, OwnerExists: true},
		{Label: "com.github.facebook.watchman", File: agent("com.github.facebook.watchman"), Scope: background.ScopeUser, Loaded: true, Running: true, PID: 3464, ProgramExists: true, OwnerExists: true},
		{Label: "com.jterrazz.brain.sync", File: agent("com.jterrazz.brain.sync"), Scope: background.ScopeUser, Loaded: true, ProgramExists: true, OwnerExists: true},
		{Label: "com.jterrazz.brain.late", File: agent("com.jterrazz.brain.late"), Scope: background.ScopeUser, Unread: true, ProgramExists: true, OwnerExists: true},
	}
}

func backgroundPages() []Fixture {
	tasks := background.Judge(backgroundJobs(), "", "/Users/jterrazz")
	return states("background",
		pending(pagebackground.List(tasks, thisMacContext(), false)),
		pagebackground.List(nil, thisMacContext(), false),
		failure("background", "background", "the job manager did not answer", "station background"),
		pagebackground.List(tasks, thisMacContext(), true))
}
