//go:build !race

package goes

import "testing"

func TestTryAllocatesNothingWhenTheCallAnswers(t *testing.T) {
	s := newSet(&record{}, nil)
	if n := testing.AllocsPerRun(1000, func() { _ = s.try("call.x", ok) }); n != 0 {
		t.Fatalf("Try allocates %v per call that answers, want 0", n)
	}
	if n := testing.AllocsPerRun(1000, func() { _ = Try("call.x", ok) }); n != 0 {
		t.Fatalf("the package's Try allocates %v per call, want 0", n)
	}
}
