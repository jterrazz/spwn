package wire

import (
	"errors"
	"testing"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

func TestAMacThatDidNotAnswerAwakeSaysWhyWithItsTokenWithheld(t *testing.T) {
	never := func(string) (model.MacRecord, bool) { return model.MacRecord{}, false }
	s := silentAwake(never, "jterrazz-mac-mini", errors.New("fetch https://bot:hunter2@proxy.example/x refused"))
	if s.Why != "fetch https://withheld@proxy.example/x refused" {
		t.Errorf("why = %q; want its credential withheld", s.Why)
	}
}
