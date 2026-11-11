package apps_test

import (
	"testing"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/core/machine"
)

func TestAServerKeepsNoAppsAMacAndAWorkspaceDo(t *testing.T) {
	for kind, want := range map[machine.Kind]bool{machine.Computer: true, machine.Workspace: true, machine.Server: false} {
		if got := apps.Applies(machine.Profile{Kind: kind}); got.OK != want || (!got.OK && got.Why == "") {
			t.Errorf("%s: %+v", kind, got)
		}
	}
}
