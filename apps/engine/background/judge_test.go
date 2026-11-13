package background_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/background"
)

func TestATaskIsAnOrphanWhenItsProgramOrItsAppIsGone(t *testing.T) {
	for _, c := range []struct {
		story string
		job   background.Job
		want  background.Group
	}{
		{"its program is missing", background.Job{Label: "com.jterrazz.brain.gone", File: "/home/me/Library/LaunchAgents/com.jterrazz.brain.gone.plist", Loaded: true, OwnerExists: true}, background.GroupOrphan},
		{"its app is gone", background.Job{Label: "com.bjango.istatmenus.installer", File: "/Library/LaunchDaemons/com.bjango.istatmenus.installer.plist", Loaded: true, ProgramExists: true}, background.GroupOrphan},
		{"its program and its app are there", background.Job{Label: "com.bjango.istatmenus.installer", File: "/Library/LaunchDaemons/com.bjango.istatmenus.installer.plist", Loaded: true, ProgramExists: true, OwnerExists: true}, background.GroupKnown},
		{"no app is known for it", background.Job{Label: "com.jterrazz.brain.x", File: "/home/me/Library/LaunchAgents/com.jterrazz.brain.x.plist", Loaded: true, ProgramExists: true, OwnerExists: true}, background.GroupUnknown},
		{"it is station's own, even with its program missing", background.Job{Label: "com.jterrazz.station.engine", File: "/home/me/Library/LaunchAgents/com.jterrazz.station.engine.plist", Loaded: true, OwnerExists: true}, background.GroupStation},
		{"it was loaded without a definition", background.Job{Label: "com.jterrazz.brain.adhoc", Loaded: true, Running: true, PID: 7, OwnerExists: true}, background.GroupUnknown},
	} {
		if got := background.Judge([]background.Job{c.job}, "", "/home/me")[0].Group; got != c.want {
			t.Errorf("%s: %s is %s, want %s", c.story, c.job.Label, got, c.want)
		}
	}
}
