package process

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

// invariant: a guest's server answers the four calls a guest's socket
// relays, as that guest, and no other: neither the engine's own calls,
// nor the registry's, nor a stream.
func TestAGuestsServerAnswersTheGuestsCallsAlone(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent})
	registry, err := call.NewRegistry([]call.Handler{own(h.d, "machines.delete", func(context.Context, *Engine, struct{}) (any, error) { return nil, nil })})
	if err != nil {
		t.Fatal(err)
	}
	s := Server(h.d, Serving{Registry: registry, Guest: "api"})
	for _, name := range []string{engine.CallPorts, engine.CallReserve, engine.CallRelease, engine.CallAgentsHeld} {
		if _, ok := s.Calls(name); !ok {
			t.Errorf("%s is not answered, want it relayed as the guest's", name)
		}
	}
	for _, name := range []string{engine.CallHello, engine.CallVMs, engine.CallStopAll, engine.CallGuestExec, engine.CallVM(engine.VMDelete), "machines.delete"} {
		if _, ok := s.Calls(name); ok {
			t.Errorf("%s is answered to a guest, want it unknown", name)
		}
	}
	if len(s.Streams) != 0 {
		t.Errorf("streams = %d, want none to a guest", len(s.Streams))
	}
}
