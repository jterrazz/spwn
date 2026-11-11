package overview_test

import (
	"testing"

	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/overview"
)

func TestOverviewIsOnEveryMachine(t *testing.T) {
	for _, kind := range []machine.Kind{machine.Computer, machine.Workspace, machine.Server} {
		if got := overview.Applies(machine.Profile{Kind: kind}); !got.OK {
			t.Errorf("Applies(%s) = %+v; want an Overview on every machine (SPEC §3.2)", kind, got)
		}
	}
}
