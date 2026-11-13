package kit

import (
	"errors"
	"testing"

	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
)

func TestAFailureWhoseRepairIsUnknownNamesTheCommandsHelp(t *testing.T) {
	got := Translate(&enginecalls.BootError{Message: "the engine did not boot", Cause: errors.New("exit 1")}, "engine start")
	if got.Message != "the engine did not boot" || got.Hint != "run station engine start --help" {
		t.Errorf("got %q / %q, want the failure with the command's help as its repair", got.Message, got.Hint)
	}
}
