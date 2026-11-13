package kit

import (
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
)

func TestAFailureWhoseRepairIsUnknownNamesTheCommandsHelp(t *testing.T) {
	got := Translate(&enginecalls.BootError{Message: "the engine did not boot", Cause: errors.New("exit 1")}, "engine start")
	if got.Message != "the engine did not boot" || got.Hint != "run station engine start --help" {
		t.Errorf("got %q / %q, want the failure with the command's help as its repair", got.Message, got.Hint)
	}
}

func TestAStoreAnOlderStationWroteAsksForStationInit(t *testing.T) {
	at := "/Users/me/.local/state/station/ledger.json"
	got := Translate(&store.UnreadableError{Path: at, Cause: &store.OlderError{Version: 1, Current: 2}}, "plan")
	if got.Hint != "run station init: it brings \""+at+"\" to version 2" {
		t.Errorf("got %q / %q, want station init as the repair", got.Message, got.Hint)
	}
}
