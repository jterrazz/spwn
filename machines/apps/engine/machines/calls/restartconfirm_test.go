package calls_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	machines "github.com/jterrazz/studio-station/apps/engine/machines/calls"
)

func TestEachKindOfTargetAsksItsOwnRestartConfirmation(t *testing.T) {
	for kind, want := range map[machine.Kind]struct {
		action confirm.Action
		level  confirm.Level
	}{
		machine.Computer:  {confirm.RestartMac, confirm.YesNo},
		machine.Server:    {confirm.RestartServer, confirm.YesNo},
		machine.Workspace: {confirm.Restart, confirm.None},
	} {
		got := machines.RestartConfirm(kind)
		if got != want.action || confirm.Of(got, kind) != want.level {
			t.Errorf("%s: %s asking %s; want %s asking %s", kind, got, confirm.Of(got, kind), want.action, want.level)
		}
	}
}
