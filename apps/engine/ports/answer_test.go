package ports_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/ports"
)

// invariant: SPEC B5.17: a port follows its VM's state: unreachable while
// it is down, starting while it boots again, not listed once it is stopped
// on purpose; a port that does not answer is never red.
func TestAPortFollowsItsVMsState(t *testing.T) {
	cases := []struct {
		halt   ports.Halt
		answer ports.Answer
		listed bool
	}{
		{ports.Runs, ports.Answers, true},
		{ports.Down, ports.Unreachable, true},
		{ports.Restarting, ports.Starting, true},
		{ports.StoppedOnPurpose, ports.Answers, false},
	}
	for _, c := range cases {
		if a, listed := ports.AnswerOf(c.halt); a != c.answer || listed != c.listed {
			t.Errorf("halt %d: %q listed %v, want %q listed %v", c.halt, a, listed, c.answer, c.listed)
		}
	}
	vm := ports.VMOf("infra", "jterrazz-mac-mini", true, []ports.Listener{{Port: 9100, Bind: "*", Process: "node_exporter"}})
	if p := ports.Classify(vm, nil)[0]; !p.Red() {
		t.Fatalf("a bridged server's wildcard port is not red while it runs")
	}
	vm.Answer = ports.Unreachable
	if p := ports.Classify(vm, nil)[0]; p.Red() || p.Answer != ports.Unreachable {
		t.Fatalf("a port of a server down is %+v", p)
	}
}
