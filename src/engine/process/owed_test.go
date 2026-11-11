package process

import (
	"context"
	"errors"
	"strings"
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

// value: an engine started over a tailnet that funnels 443 from 40000 for
// a VM deleted while it was down, whose reconciliation could not read the
// tailnet; db, local, running.
func staleFunnel(t *testing.T, failShared int, kept ...ports.Reservation) (harness, *flakyTailnet, *[]string) {
	t.Helper()
	h := newHarness(t, model.VMRecord{Name: "db", Lifetime: model.Persistent})
	tn := &flakyTailnet{keptTailnet: funneledFrom40000(), failShared: failShared}
	h.d.tn = tn
	h.d.o.Rights = rightsTo(catalog.AudienceLocal, 5432)
	var log []string
	h.d.note = func(line string) { log = append(log, line) }
	h.d.reservations = kept
	ctx := t.Context()
	h.d.refreshSelf(ctx)
	h.d.adopt(ctx)
	h.d.mu.Lock()
	h.d.vms["db"].state = engine.StateRunning
	h.d.mu.Unlock()
	h.d.reconcileServe(ctx)
	return h, tn, &log
}

func reservedPort(h harness, vm string, port int) int {
	h.d.mu.Lock()
	defer h.d.mu.Unlock()
	l, i := ports.Find(h.d.reservations, vm, port)
	if i < 0 {
		return 0
	}
	return l.MacPort
}

// invariant: a port of the Mac a stale Funnel entry carries from is given
// to no new share, and the reconciliation that could not read the tailnet
// at the start reads it again at the next tick and removes the entry.
func TestAStaleFunnelEntryCarriesNoNewShare(t *testing.T) {
	h, tn, _ := staleFunnel(t, 1)
	ctx := t.Context()
	h.d.listen(ctx, "db", 5432, "0.0.0.0", "postgres")
	if got := reservedPort(h, "db", 5432); got == 0 || got == 40000 {
		t.Errorf("db reserved %d, want a port the stale Funnel entry does not carry", got)
	}
	h.d.tickShares(ctx)
	if got := tn.dump(); got != "" {
		t.Errorf("tailnet after the next tick = %s, want the stale entry gone", got)
	}
}

// invariant: a reservation kept on a port a tailnet entry not its own
// carries from moves off it, and says so.
func TestAReservationMovesOffAPortAStaleEntryCarries(t *testing.T) {
	no := false
	h, tn, _ := staleFunnel(t, 1, ports.Reservation{VM: "db", GuestPort: 5432, MacPort: 40000, To: catalog.AudienceLocal, By: ports.ByRights, HTTP: &no})
	ctx := t.Context()
	if got := tn.dump(); got != "443/<-40000 funnel=true" {
		t.Fatalf("tailnet before the listen = %s %v, want the stale entry", got, tn.said())
	}
	h.d.listen(ctx, "db", 5432, "0.0.0.0", "postgres")
	if got := reservedPort(h, "db", 5432); got == 40000 {
		t.Errorf("db bound 40000, which the stale Funnel entry carries to the internet: %s", tn.dump())
	}
	h.d.mu.Lock()
	detail := h.d.svcs[key("db", 5432)].detail
	h.d.mu.Unlock()
	if detail != "reserved port 40000 was carried on the tailnet: now 40001" {
		t.Errorf("detail = %q, want the move said", detail)
	}
}

// invariant: while Serve is on and the tailnet was never read, no port is
// given out; the tick that reads it removes the stale entry and shares the
// port refused meanwhile, the failed reads said once in the engine's log.
func TestATailnetNeverReadGivesOutNoPortUntilATickReadsIt(t *testing.T) {
	h, tn, log := staleFunnel(t, 3)
	ctx := t.Context()
	h.d.listen(ctx, "db", 5432, "0.0.0.0", "postgres")
	if got := reservedPort(h, "db", 5432); got != 0 {
		t.Fatalf("db reserved %d while the tailnet was never read, want none", got)
	}
	h.d.tickShares(ctx)
	h.d.tickShares(ctx)
	if got := tn.dump(); got != "" {
		t.Errorf("tailnet after the tick that read it = %s, want the stale entry gone", got)
	}
	if got := reservedPort(h, "db", 5432); got == 0 {
		t.Errorf("db is not reserved after the tick that read the tailnet")
	}
	said := 0
	for _, l := range *log {
		if strings.Contains(l, "could not be read") {
			said++
		}
	}
	if said != 1 {
		t.Errorf("the failed reads were said %d times, want once: %v", said, *log)
	}
}
