package cliproject

import "testing"

func TestTheAskNeedsStdinAndStderrBothTerminals(t *testing.T) {
	for _, c := range []struct {
		stdin, stderr, want bool
	}{
		{true, true, true},
		{true, false, false},
		{false, true, false},
		{false, false, false},
	} {
		isTerminal := func(fd int) bool { return map[int]bool{0: c.stdin, 2: c.stderr}[fd] }
		if got := bothTerminals(isTerminal, 0, 2); got != c.want {
			t.Errorf("stdin terminal %v, stderr terminal %v: asked %v, want %v", c.stdin, c.stderr, got, c.want)
		}
	}
}
