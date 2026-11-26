package process

import (
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"

	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
)

func TestACommandEndedPastTheMostKeptIsTheEnginesCodedFailureNamingItsVM(t *testing.T) {
	printed := engine.GuestRan{Stdout: "the first bytes"}
	ran, err := pastMost("linux-1")(printed, &contract.OutputTooLargeError{VM: "linux-1"})
	var op *enginecalls.OpError
	if !errors.As(err, &op) || op.Code != enginecalls.GuestOutputTooLarge {
		t.Fatalf("err = %v; want %s", err, enginecalls.GuestOutputTooLarge)
	}
	if op.Message != "linux-1's command printed more than 8 MiB, the most a command's output is kept, so it was ended" || op.Hint != "station machines linux-1" {
		t.Errorf("the failure reads %q, %q", op.Message, op.Hint)
	}
	if ran != printed {
		t.Errorf("the output answered is %+v, want what it printed up to the most", ran)
	}
}

func TestAnyOtherEndOfACommandLeavesAsItCame(t *testing.T) {
	broke := errors.New("the connection broke")
	if _, err := pastMost("linux-1")(engine.GuestRan{}, broke); !errors.Is(err, broke) {
		t.Errorf("err = %v; want the break itself", err)
	}
}
