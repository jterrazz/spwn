package fail_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
)

func TestEachErrorTypeExitsWithItsCode(t *testing.T) {
	for typ, want := range map[fail.Type]fail.Exit{
		fail.Refused:           fail.ExitUsage,
		fail.NotFound:          fail.ExitUsage,
		fail.NotForThisMachine: fail.ExitUsage,
		fail.NeedsConfirmation: fail.ExitUsage,
		fail.NeedsTerminal:     fail.ExitUsage,
		fail.Declined:          fail.ExitFailure,
		fail.Unreachable:       fail.ExitFailure,
		fail.EngineStopped:     fail.ExitFailure,
		fail.FarTooOld:         fail.ExitFailure,
		fail.Failed:            fail.ExitFailure,
		fail.Interrupted:       fail.ExitInterrupted,
	} {
		if got := fail.ExitOf(fail.New(typ, "x.y", "infra", nil)); got != want {
			t.Errorf("an error of type %s exits %d, want %d", typ, got, want)
		}
	}
}
