package process

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

type noteLog struct {
	mu    sync.Mutex
	lines []string
}

func (n *noteLog) add(line string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.lines = append(n.lines, line)
}

func (n *noteLog) all() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]string(nil), n.lines...)
}

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
	return funneledAPIOver(t, nil)
}

// value: funneledAPI with the ports of the Mac busy in mac, set before the
// engine's goroutines read it; nil leaves the harness's own.
func funneledAPIOver(t *testing.T, busy map[int]bool) (harness, *flakyTailnet) {
	t.Helper()
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent}, model.VMRecord{Name: "db", Lifetime: model.Persistent})
	tn := &flakyTailnet{keptTailnet: funneledFrom40000()}
	h.d.tn = tn
	if busy != nil {
		h.d.h = &busyMac{fakeMac: h.mac, busy: busy}
	}
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
	h, tn := funneledAPIOver(t, map[int]bool{40000: true})
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
func staleFunnel(t *testing.T, failShared int, kept ...ports.Reservation) (harness, *flakyTailnet, *noteLog) {
	t.Helper()
	h := newHarness(t, model.VMRecord{Name: "db", Lifetime: model.Persistent})
	tn := &flakyTailnet{keptTailnet: funneledFrom40000(), failShared: failShared}
	h.d.tn = tn
	h.d.o.Rights = rightsTo(catalog.AudienceLocal, 5432)
	log := &noteLog{}
	h.d.note = log.add
	h.d.reservations = kept
	ctx := t.Context()
	h.d.refreshSelf(ctx)
	h.d.adopt(ctx)
	h.d.mu.Lock()
	h.d.vms["db"].state = engine.StateRunning
	h.d.mu.Unlock()
	h.d.reconcileServe(ctx)
	return h, tn, log
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
	lines := log.all()
	for _, l := range lines {
		if strings.Contains(l, "could not be read") {
			said++
		}
	}
	if said != 1 {
		t.Errorf("the failed reads were said %d times, want once: %v", said, lines)
	}
}

// invariant: a reservation kept across a restart whose share was refused
// while the tailnet was never read is shared at the tick that reads it.
func TestAKeptReservationRefusedUnreadIsSharedAtTheTickThatReadsTheTailnet(t *testing.T) {
	no := false
	h, tn, _ := staleFunnel(t, 1000, ports.Reservation{VM: "db", GuestPort: 5432, MacPort: 40001, To: catalog.AudienceLocal, By: ports.ByRights, HTTP: &no})
	ctx := t.Context()
	h.d.listen(ctx, "db", 5432, "0.0.0.0", "postgres")
	tn.failShared = 0
	for range 5 {
		h.d.tickShares(ctx)
	}
	h.d.mu.Lock()
	s := h.d.svcs[key("db", 5432)]
	phase, listeners := s.st.Phase, len(s.listeners)
	h.d.mu.Unlock()
	if phase != ports.PhaseShared || listeners == 0 {
		t.Errorf("kept reservation after the ticks that read the tailnet: phase=%s listeners=%d, want shared", phase, listeners)
	}
}

// invariant: an entry this engine served goes at its release even when the
// tailnet cannot be read, never left for a later tick.
func TestAServedEntryGoesWhenTheTailnetCannotBeRead(t *testing.T) {
	h, tn := funneledAPI(t)
	ctx := t.Context()
	tn.failShared = 1
	if _, err := h.d.release(ctx, "api", 3000); err != nil {
		t.Fatal(err)
	}
	if got := tn.dump(); got != "" {
		t.Errorf("tailnet after the release = %s, want no entry", got)
	}
}

// invariant: the port of the Mac an owed entry carries from is given to no
// share while the tailnet cannot be read, though the last reading, made
// before that entry was served, does not list it.
func TestAnOwedPortIsGivenToNoShareWhileTheTailnetIsUnread(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent}, model.VMRecord{Name: "db", Lifetime: model.Persistent})
	tn := &flakyTailnet{keptTailnet: &keptTailnet{fakeTailnet: fakeTailnet{sharing: ports.SharingState{On: true, DNSName: "x.tail1234.ts.net", Backend: ports.Running}}}}
	h.d.tn = tn
	h.d.o.Rights = func(ctx context.Context, vm string) (*catalog.PortRights, error) {
		if vm == "db" {
			return rightsTo(catalog.AudienceLocal, 5432)(ctx, vm)
		}
		return rightsTo(catalog.AudiencePublic, 3000)(ctx, vm)
	}
	ctx := t.Context()
	h.d.refreshSelf(ctx)
	h.d.adopt(ctx)
	h.d.mu.Lock()
	for _, v := range h.d.vms {
		v.state = engine.StateRunning
	}
	h.d.mu.Unlock()
	h.d.reconcileServe(ctx)
	h.d.listen(ctx, "api", 3000, "0.0.0.0", "node")
	if got := tn.dump(); got != "443<-40000 funnel=true" {
		t.Fatalf("tailnet after api's share = %s, want 443 funneled from 40000", got)
	}
	tn.failShared, tn.failUnshare = 3, 1
	if _, err := h.d.release(ctx, "api", 3000); err != nil {
		t.Fatal(err)
	}
	h.d.listen(ctx, "db", 5432, "0.0.0.0", "postgres")
	if got := reservedPort(h, "db", 5432); got == 0 || got == 40000 {
		t.Errorf("db reserved %d, want a port the owed Funnel entry does not carry", got)
	}
}

// invariant: a share an owed entry held, a private path on a port that
// entry still funnels, is served at the tick that removes the entry.
func TestAShareAnOwedEntryHeldIsServedAtTheTickThatRemovesIt(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent}, model.VMRecord{Name: "pub", Lifetime: model.Persistent})
	pol := &heldPolicy{g: &ports.PortsGroup{HTTPS: "port"}}
	h.d.st.Ports = pol
	tn := &flakyTailnet{keptTailnet: &keptTailnet{fakeTailnet: fakeTailnet{sharing: ports.SharingState{On: true, DNSName: "x.tail1234.ts.net", Backend: ports.Running}}}}
	h.d.tn = tn
	h.d.o.Rights = func(ctx context.Context, vm string) (*catalog.PortRights, error) {
		if vm == "pub" {
			return rightsTo(catalog.AudiencePublic, 3000)(ctx, vm)
		}
		return rightsTo(catalog.AudiencePrivate, 5173)(ctx, vm)
	}
	h.d.o.Ports = []catalog.PackagePort{{PortHint: catalog.PortHint{Port: 5173, HTTP: true}}, {PortHint: catalog.PortHint{Port: 3000, HTTP: true}}}
	ctx := t.Context()
	h.d.refreshSelf(ctx)
	h.d.reconcileServe(ctx)
	h.d.tickShares(ctx)
	h.d.loadRights(ctx, "api")
	h.d.loadRights(ctx, "pub")
	h.d.mu.Lock()
	for _, vm := range []string{"api", "pub"} {
		if v := h.d.vms[vm]; v != nil {
			v.state = engine.StateRunning
		}
	}
	h.d.mu.Unlock()
	h.d.listen(ctx, "api", 5173, "0.0.0.0", "vite")
	h.d.listen(ctx, "pub", 3000, "0.0.0.0", "node")
	if got := tn.dump(); got != "40000<-40000 funnel=false; 443<-40001 funnel=true" {
		t.Fatalf("tailnet in port mode = %s", got)
	}
	tn.failUnshare = 2
	pol.g = &ports.PortsGroup{HTTPS: "path"}
	h.d.tickShares(ctx)
	h.d.mu.Lock()
	api := h.d.svcs[key("api", 5173)]
	held := !api.served
	h.d.mu.Unlock()
	if !held {
		t.Fatalf("api served while 443 still funnels the owed entry: %s", tn.dump())
	}
	h.d.tickShares(ctx)
	h.d.mu.Lock()
	served := api.served
	h.d.mu.Unlock()
	if got := tn.dump(); !served || got != "443/api/5173<-40000 funnel=false; 8443<-40002 funnel=true" {
		t.Errorf("after the tick that removed the owed entries: api served %v, tailnet %s; want api's path served and 443 unfunneled", served, got)
	}
}

// invariant: Serve turned off and on again reads the tailnet again: a
// reconciliation that cannot read it then is tried again at the next tick.
func TestServeTurnedOnAgainReconcilesUntilTheTailnetIsRead(t *testing.T) {
	h := newHarness(t)
	tn := &flakyTailnet{keptTailnet: &keptTailnet{fakeTailnet: fakeTailnet{sharing: ports.SharingState{On: true, DNSName: "x.tail1234.ts.net", Backend: ports.Running}}}}
	h.d.tn = tn
	ctx := t.Context()
	h.d.refreshSelf(ctx)
	h.d.reconcileServe(ctx)
	tn.mu.Lock()
	tn.sharing.On = false
	tn.mu.Unlock()
	h.d.refreshSelf(ctx)
	stale := funneledFrom40000()
	tn.mu.Lock()
	tn.entries, tn.funnel, tn.sharing.On = stale.entries, stale.funnel, true
	tn.mu.Unlock()
	tn.failShared = 1
	h.d.refreshSelf(ctx)
	h.d.tickShares(ctx)
	if got := tn.dump(); got != "" {
		t.Errorf("tailnet after the tick = %s, want the stale entry gone", got)
	}
}

// invariant: the last reading of the tailnet, by any read, stands while
// it cannot be read: a port an entry the owner made carries from is given
// to no share.
func TestThePortOfAnEntryLastReadIsGivenToNoShareWhileTheTailnetIsUnread(t *testing.T) {
	h, tn, _ := staleFunnel(t, 0)
	ctx := t.Context()
	h.d.reconcileServe(ctx)
	tn.mu.Lock()
	tn.entries = []ports.Share{{Port: 9000, Reserved: 40000, Scheme: ports.SchemeTCP, Target: "tcp://127.0.0.1:40000"}}
	tn.funnel = nil
	tn.mu.Unlock()
	h.d.mu.Lock()
	_, err := h.d.sharedLocked(ctx)
	h.d.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	tn.failShared = 2
	h.d.listen(ctx, "db", 5432, "0.0.0.0", "postgres")
	if got := reservedPort(h, "db", 5432); got == 0 || got == 40000 {
		t.Errorf("db reserved %d, want a port the owner's entry, as last read, does not carry", got)
	}
}

// invariant: an owed entry that its reservation serves again, the HTTPS
// mode switched back before the removal went, is kept: the tick removes
// only what no reservation justifies.
func TestAnOwedEntryServedAgainIsKept(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent})
	pol := &heldPolicy{g: &ports.PortsGroup{HTTPS: "path"}}
	h.d.st.Ports = pol
	tn := &flakyTailnet{keptTailnet: &keptTailnet{fakeTailnet: fakeTailnet{sharing: ports.SharingState{On: true, DNSName: "x.tail1234.ts.net", Backend: ports.Running}}}}
	h.d.tn = tn
	h.d.o.Rights = rightsTo(catalog.AudiencePrivate, 5173)
	h.d.o.Ports = []catalog.PackagePort{{PortHint: catalog.PortHint{Port: 5173, HTTP: true}}}
	ctx := t.Context()
	h.d.refreshSelf(ctx)
	h.d.adopt(ctx)
	h.d.reconcileServe(ctx)
	h.d.tickShares(ctx)
	h.d.mu.Lock()
	h.d.vms["api"].state = engine.StateRunning
	h.d.mu.Unlock()
	h.d.listen(ctx, "api", 5173, "0.0.0.0", "vite")
	pol.g = &ports.PortsGroup{HTTPS: "port"}
	tn.failUnshare = 1
	h.d.tickShares(ctx)
	if got := tn.dump(); got != "40000<-40000 funnel=false; 443/api/5173<-40000 funnel=false" {
		t.Fatalf("tailnet after the failed switch to port = %s, want the path owed beside the port", got)
	}
	pol.g = &ports.PortsGroup{HTTPS: "path"}
	tn.failUnshare = 1
	h.d.tickShares(ctx)
	h.d.tickShares(ctx)
	if got := tn.dump(); got != "443/api/5173<-40000 funnel=false" {
		t.Errorf("tailnet after the switch back to path = %s, want api's path served", got)
	}
}
