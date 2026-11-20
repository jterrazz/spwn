package process

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"

	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
)

type addressed struct{ *fakeHypervisor }

func (addressed) Address(context.Context, string, time.Duration) (netip.Addr, error) {
	return netip.MustParseAddr("192.168.64.3"), nil
}

type dialing struct {
	err   error
	guest Guest
}

func (g *dialing) Dial(context.Context, string, netip.AddrPort) (Guest, error) {
	if g.err != nil {
		return nil, g.err
	}
	return g.guest, nil
}

type unattached struct{ Guest }

func (unattached) Attach(context.Context, string) (Session, error) {
	return nil, errors.New("no session in a module test")
}
func (unattached) Close() error { return nil }

var errRefusedByLocalNetwork = fmt.Errorf("%w: dial tcp 192.168.64.3:22: connect", contract.ErrLocalNetwork)

func blockedHarness(t *testing.T) (harness, *dialing, *vmState) {
	t.Helper()
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent, Want: model.WantRunning})
	g := &dialing{err: errRefusedByLocalNetwork}
	h.d.hv, h.d.guests = addressed{h.hv}, g
	h.hv.outlived("api")
	v, _ := h.d.find("api")
	v.run, _ = h.hv.Adopt(t.Context(), "api")
	v.state, v.guest = engine.StateRunning, engine.GuestConnecting
	return h, g, v
}

func blockOf(h harness) model.GuestBlock {
	for _, view := range h.d.vmViews() {
		if view.Name == "api" {
			return view.Blocked
		}
	}
	return "absent"
}

func TestADialItsMacRefusesForLocalNetworkBlocksTheVMUntilOneConnects(t *testing.T) {
	h, g, v := blockedHarness(t)
	h.d.guestOnce(t.Context(), v, v.gen)
	if got := blockOf(h); got != model.LocalNetworkBlock {
		t.Fatalf("after a dial refused for Local Network: blocked %q, want %q", got, model.LocalNetworkBlock)
	}
	g.err, g.guest = nil, unattached{}
	h.d.guestOnce(t.Context(), v, v.gen)
	if got := blockOf(h); got != "" {
		t.Errorf("after a dial that connected: blocked %q, want none", got)
	}
}

func TestADialRefusedForAnotherReasonIsNoBlock(t *testing.T) {
	h, g, v := blockedHarness(t)
	h.d.guestOnce(t.Context(), v, v.gen)
	g.err = errors.New("dial tcp 192.168.64.3:22: connect: connection refused")
	h.d.guestOnce(t.Context(), v, v.gen)
	if got := blockOf(h); got != "" {
		t.Errorf("after a refused dial: blocked %q, want none", got)
	}
}

func TestARunThatEndsTakesItsBlockWithIt(t *testing.T) {
	h, _, v := blockedHarness(t)
	h.d.guestOnce(t.Context(), v, v.gen)
	h.d.disconnect(v)
	if got := blockOf(h); got != "" {
		t.Errorf("after its run ended: blocked %q, want none", got)
	}
}

func TestARunItsMacBlocksAnswersLocalNetworkAfterAMinuteNamingTheMac(t *testing.T) {
	h, _, v := blockedHarness(t)
	h.d.guestOnce(t.Context(), v, v.gen)
	began := h.clock.At
	err := h.d.awaitGuest(t.Context(), "api")
	var op *enginecalls.OpError
	if !errors.As(err, &op) || op.Code != enginecalls.LocalNetwork {
		t.Fatalf("a guest its Mac blocks: %v, want local_network", err)
	}
	if want := "x has not approved Machines for Local Network"; op.Message != want {
		t.Errorf("message %q, want %q", op.Message, want)
	}
	if want := "approve Machines on x's screen, or in System Settings > Privacy & Security > Local Network"; op.Hint != want {
		t.Errorf("hint %q, want %q", op.Hint, want)
	}
	if waited := h.clock.At.Sub(began); waited < 60*time.Second {
		t.Errorf("local_network after %s, want the minute a guest has to connect", waited)
	}
}
