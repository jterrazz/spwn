package words_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/words"
)

func TestAnEngineRefusalPrintsTheWordsItCarriedNamingTheVMAtHand(t *testing.T) {
	carried := &fail.Error{
		Code:  enginecalls.VMStopped,
		Cause: &fail.Said{Message: "api is stopped", Hint: "station machines start api"},
	}
	message, hint := words.Codes()[enginecalls.VMStopped](carried)
	if message != "api is stopped" || hint != "station machines start api" {
		t.Errorf("a carried refusal reads %q, %q", message, hint)
	}

	bare := &fail.Error{Code: enginecalls.VMStopped}
	if message, _ := words.Codes()[enginecalls.VMStopped](bare); message != "the VM is stopped" {
		t.Errorf("a refusal with no words of its own reads %q", message)
	}
}
