package fake

import (
	"context"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestASampleTheWorldScriptsToPanicLeavesTheWorldFree(t *testing.T) {
	w := &world.World{ThisMac: "mbp"}
	w.Script = []world.Fault{{Call: "Meter.Sample", Times: 1, Panic: true}}
	m := MeterOf(w, "mbp")
	broke := func() (broke bool) {
		defer func() { broke = recover() != nil }()
		_, _ = m.Sample(t.Context())
		return false
	}()
	if !broke {
		t.Fatal("Meter.Sample did not panic: the test proves nothing")
	}
	done := make(chan struct{})
	go func() {
		_, _ = m.Sample(t.Context())
		close(done)
	}()
	bound, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	select {
	case <-done:
	case <-bound.Done():
		t.Fatal("the next sample after a world panic did not answer: the fake kept the world locked")
	}
}
