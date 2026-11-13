package activity_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

func TestActivityIsOnEveryMachine(t *testing.T) {
	for _, kind := range []machine.Kind{machine.Computer, machine.Workspace, machine.Server} {
		if got := activity.Applies(machine.Profile{Kind: kind}); !got.OK {
			t.Errorf("Applies(%s) = %+v; want Activity on every machine (SPEC §3.2)", kind, got)
		}
	}
}
