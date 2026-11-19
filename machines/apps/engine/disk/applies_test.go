package disk_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/disk"
)

func TestAServerHasNoDiskSectionAMacAndAWorkspaceDo(t *testing.T) {
	for kind, want := range map[machine.Kind]bool{machine.Computer: true, machine.Workspace: true, machine.Server: false} {
		if got := disk.Applies(machine.Profile{Kind: kind}); got.OK != want || (!got.OK && got.Why == "") {
			t.Errorf("%s: %+v", kind, got)
		}
	}
}
