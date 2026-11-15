package process

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/ports"
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

// invariant: the engine's socket refuses an audience of none of the three
// words before it reserves anything, as ports.share refuses it.
func TestAReserveOfAnUnknownAudienceIsRefusedOnTheSocket(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent})
	for _, c := range calls(h.d, nil) {
		if c.Meta().Name != engine.CallReserve {
			continue
		}
		_, err := c.Invoke(t.Context(), machine.Profile{}, json.RawMessage(`{"vm":"api","guest_port":3000,"to":"everyone"}`), nil)
		var refused *fail.Error
		if !errors.As(err, &refused) || refused.Code != ports.CodeUnknownAudience {
			t.Fatalf("reserve to everyone: %v, want %s", err, ports.CodeUnknownAudience)
		}
		if len(h.d.reservations) != 0 {
			t.Errorf("reservations %v, want none made", h.d.reservations)
		}
		return
	}
	t.Fatal("no engine.reserve served")
}

// invariant: the engine's hello lists the abilities it was started with,
// so a caller asks an audience only of an engine that reads one.
func TestTheEnginesHelloListsItsAbilities(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent})
	h.d.o.Abilities = engine.Abilities()
	for _, c := range calls(h.d, nil) {
		if c.Meta().Name != engine.CallHello {
			continue
		}
		out, err := c.Invoke(t.Context(), machine.Profile{}, nil, nil)
		var hello engine.Hello
		if err == nil {
			err = json.Unmarshal(out, &hello)
		}
		if err != nil || len(hello.Abilities) != 1 || hello.Abilities[0] != engine.AbilityReserveTo {
			t.Fatalf("hello = %s, %v; want it to list %s", out, err, engine.AbilityReserveTo)
		}
		return
	}
	t.Fatal("no engine.hello served")
}
