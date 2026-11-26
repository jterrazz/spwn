package process

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/approvals"
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
	grant approvals.State
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

var errNoRoute = &contract.NoRouteError{Err: errors.New("dial tcp 192.168.64.3:22: connect: no route to host")}

func blockedHarness(t *testing.T) (harness, *dialing, *vmState) {
	t.Helper()
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent, Want: model.WantRunning})
	g := &dialing{err: errNoRoute, grant: approvals.Refused}
	h.d.hv, h.d.guests = addressed{h.hv}, g
	h.d.o.LocalNetwork = func(context.Context) approvals.State { return g.grant }
	h.clock.At = h.clock.At.Add(approvals.LocalNetworkSettling)
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

func TestADialWithNoRouteWhileTheGrantReadsApprovedIsNoBlockAndSaysOnlyWhatItMet(t *testing.T) {
	for name, grant := range map[string]approvals.State{"approved": approvals.Approved, "unreadable": approvals.Unknown} {
		t.Run(name, func(t *testing.T) {
			h, g, v := blockedHarness(t)
			g.grant = grant
			h.d.guestOnce(t.Context(), v, v.gen)
			if got := blockOf(h); got != "" {
				t.Errorf("a dial with no route, the grant reading %q: blocked %q, want none", grant, got)
			}
			for _, view := range h.d.vmViews() {
				if want := "dial tcp 192.168.64.3:22: connect: no route to host"; view.Name == "api" && view.GuestError != want {
					t.Errorf("guest error %q, want the dial's own %q", view.GuestError, want)
				}
			}
		})
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
	if want := "x has not approved station for Local Network"; op.Message != want {
		t.Errorf("message %q, want %q", op.Message, want)
	}
	if want := "approve station on x's screen, or in System Settings > Privacy & Security > Local Network"; op.Hint != want {
		t.Errorf("hint %q, want %q", op.Hint, want)
	}
	if waited := h.clock.At.Sub(began); waited < 60*time.Second {
		t.Errorf("local_network after %s, want the minute a guest has to connect", waited)
	}
}
