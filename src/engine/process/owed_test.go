package process

import (
	"context"
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/engine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/ports"
)

// value: a kept tailnet whose next failUnshare unshares and failShared
// reads fail, as a serve that times out.
type flakyTailnet struct {
	*keptTailnet
	failUnshare int
	failShared  int
}

func (f *flakyTailnet) Unshare(ctx context.Context, s ports.Share) error {
	if f.failUnshare > 0 {
		f.failUnshare--
		return errors.New("serve: timeout")
	}
	return f.keptTailnet.Unshare(ctx, s)
}

func (f *flakyTailnet) Shared(ctx context.Context) ([]ports.Share, error) {
	if f.failShared > 0 {
		f.failShared--
		return nil, errors.New("serve status: timeout")
	}
	return f.keptTailnet.Shared(ctx)
}

func rightsTo(to catalog.Audience, port int) func(context.Context, string) (*catalog.PortRights, error) {
	return func(context.Context, string) (*catalog.PortRights, error) {
		return &catalog.PortRights{Open: catalog.PortSet{{From: port, To: port}}, To: to}, nil
	}
}

// value: an engine started over a tailnet that funnels 443 from 40000, the
// entry of api's public reservation there, api running.
func funneledAPI(t *testing.T) (harness, *flakyTailnet) {
	t.Helper()
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent}, model.VMRecord{Name: "db", Lifetime: model.Persistent})
	tn := &flakyTailnet{keptTailnet: funneledFrom40000()}
	h.d.tn = tn
	h.d.o.Rights = func(ctx context.Context, vm string) (*catalog.PortRights, error) {
		if vm == "db" {
			return rightsTo(catalog.AudienceLocal, 5432)(ctx, vm)
		}
		return rightsTo(catalog.AudiencePublic, 3000)(ctx, vm)
	}
	yes := true
	h.d.reservations = []ports.Reservation{{VM: "api", GuestPort: 3000, MacPort: 40000, To: catalog.AudiencePublic, PublicPort: 443, By: ports.ByRights, HTTP: &yes, ReservedAt: h.clock.At, LastSeenAt: h.clock.At}}
	ctx := t.Context()
	h.d.refreshSelf(ctx)
	h.d.adopt(ctx)
	h.d.mu.Lock()
	for _, v := range h.d.vms {
		v.state = engine.StateRunning
	}
	h.d.mu.Unlock()
	h.d.reconcileServe(ctx)
	return h, tn
}

// invariant: a reservation moved off its busy port whose old Funnel entry
// fails to go replaces that entry with its own, and its release leaves
// nothing on the internet.
func TestABusyMoveWhoseUnshareFailsLeavesNoFunnelFromItsOldPort(t *testing.T) {
	h, tn := funneledAPI(t)
	h.d.h = &busyMac{fakeMac: h.mac, busy: map[int]bool{40000: true}}
	ctx := t.Context()
	tn.failUnshare = 1
	h.d.listen(ctx, "api", 3000, "0.0.0.0", "node")
	if got := tn.dump(); got != "443<-40001 funnel=true" {
		t.Errorf("tailnet after the move = %s, want 443 funneled from 40001 only", got)
	}
	if _, err := h.d.release(ctx, "api", 3000); err != nil {
		t.Fatal(err)
	}
	h.d.tickShares(ctx)
	if got := tn.dump(); got != "" {
		t.Errorf("tailnet after the release = %s, want no entry", got)
	}
}

// invariant: an unshare that fails is owed: the port of the Mac its entry
// carries from is given to no other reservation, and the next tick removes
// the entry.
func TestAFailedUnshareIsTriedAgainAtTheNextTick(t *testing.T) {
	h, tn := funneledAPI(t)
	ctx := t.Context()
	tn.failUnshare = 1
	if _, err := h.d.release(ctx, "api", 3000); err != nil {
		t.Fatal(err)
	}
	if got := tn.dump(); got != "443/<-40000 funnel=true" {
		t.Fatalf("tailnet after the failed release = %s, want the entry still there", got)
	}
	h.d.listen(ctx, "db", 5432, "0.0.0.0", "postgres")
	h.d.mu.Lock()
	l, _ := ports.Find(h.d.reservations, "db", 5432)
	h.d.mu.Unlock()
	if l.MacPort == 40000 {
		t.Errorf("db was given 40000, which the owed Funnel entry still carries to the internet")
	}
	h.d.tickShares(ctx)
	if got := tn.dump(); got != "" {
		t.Errorf("tailnet after the next tick = %s, want no entry", got)
	}
}

// invariant: an entry no reservation justifies that the reconciliation
// fails to remove goes at the next tick.
func TestAnEntryTheReconciliationFailsToRemoveGoesAtTheNextTick(t *testing.T) {
	h := newHarness(t)
	tn := &flakyTailnet{keptTailnet: funneledFrom40000(), failUnshare: 1}
	h.d.tn = tn
	ctx := t.Context()
	h.d.refreshSelf(ctx)
	h.d.reconcileServe(ctx)
	if got := tn.dump(); got != "443/<-40000 funnel=true" {
		t.Fatalf("tailnet after the failed reconciliation = %s, want the entry still there", got)
	}
	h.d.tickShares(ctx)
	if got := tn.dump(); got != "" {
		t.Errorf("tailnet after the next tick = %s, want no entry", got)
	}
}
