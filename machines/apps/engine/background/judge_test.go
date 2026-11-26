package background_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/background"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

func TestATaskIsAnOrphanWhenItsProgramOrItsAppIsGone(t *testing.T) {
	for _, c := range []struct {
		story string
		job   background.Job
		want  background.Group
	}{
		{"its program is missing", background.Job{Label: "com.example.brain.gone", File: "/home/me/Library/LaunchAgents/com.example.brain.gone.plist", Loaded: true, OwnerExists: true}, background.GroupOrphan},
		{"its app is gone", background.Job{Label: "com.bjango.istatmenus.installer", File: "/Library/LaunchDaemons/com.bjango.istatmenus.installer.plist", Loaded: true, ProgramExists: true}, background.GroupOrphan},
		{"its program and its app are there", background.Job{Label: "com.bjango.istatmenus.installer", File: "/Library/LaunchDaemons/com.bjango.istatmenus.installer.plist", Loaded: true, ProgramExists: true, OwnerExists: true}, background.GroupKnown},
		{"no app is known for it", background.Job{Label: "com.example.brain.x", File: "/home/me/Library/LaunchAgents/com.example.brain.x.plist", Loaded: true, ProgramExists: true, OwnerExists: true}, background.GroupUnknown},
		{"it is Machines' own, even with its program missing", background.Job{Label: "com.jterrazz.station.engine", File: "/home/me/Library/LaunchAgents/com.jterrazz.station.engine.plist", Loaded: true, OwnerExists: true}, background.GroupStation},
		{"it is Spwn.app's engine, loaded from the bundle", background.Job{Label: "com.jterrazz.spwn.engine", Loaded: true, Running: true, PID: 13629, OwnerExists: true}, background.GroupStation},
		{"it is Spwn.app's agent, loaded from the bundle", background.Job{Label: "com.jterrazz.spwn.agent", Loaded: true, Running: true, PID: 9006, OwnerExists: true}, background.GroupStation},
		{"it was loaded without a definition", background.Job{Label: "com.example.brain.adhoc", Loaded: true, Running: true, PID: 7, OwnerExists: true}, background.GroupUnknown},
	} {
		if got := background.Judge([]background.Job{c.job}, "", "/home/me")[0].Group; got != c.want {
			t.Errorf("%s: %s is %s, want %s", c.story, c.job.Label, got, c.want)
		}
	}
}

func TestAnOwnJobThatIsNotThisSpwnsIsLeftOverAndRemovedByHand(t *testing.T) {
	job := background.Job{Label: "com.jterrazz.spwn.agent", Scope: background.ScopeUser, Loaded: true, Running: true, PID: 9006, OwnerExists: true, ProgramExists: true,
		Program: "/private/tmp/old/Spwn.app/Contents/MacOS/Spwn Engine", Foreign: true, Removal: "the removal of com.jterrazz.spwn.agent"}
	task := background.Judge([]background.Job{job}, "", "/home/me")[0]
	if task.Group != background.GroupOrphan || !task.Foreign || task.Removal != job.Removal {
		t.Fatalf("an own job of another Spwn is %+v", task)
	}
	found := background.Findings([]background.Task{task}, machine.Ref{})
	if len(found) != 1 || found[0].Text != "com.jterrazz.spwn.agent runs /private/tmp/old/Spwn.app/Contents/MacOS/Spwn Engine, not this Spwn's" || found[0].Fix.Command != job.Removal || !found[0].Fix.InTerminal {
		t.Errorf("its finding is %+v", found)
	}
}
