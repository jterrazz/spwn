package process

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/engine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/ports"
)

type heldPolicy struct{ g *ports.PortsGroup }

func (p *heldPolicy) Get() (*ports.PortsGroup, error) { return p.g, nil }

// value: a tailnet that keeps what it is told, Funnel per port as
// tailscale keeps it: a port funneled while any entry of it remains, and
// an https entry with no path at the root, /.
type keptTailnet struct {
	fakeTailnet
	entries []ports.Share
	funnel  map[int]bool
}

func samePath(a, b ports.Share) bool {
	return a.Port == b.Port && cmp.Or(a.Path, "/") == cmp.Or(b.Path, "/")
}

func (f *keptTailnet) Share(_ context.Context, s ports.Share) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, said("share", s))
	f.entries = slices.DeleteFunc(f.entries, func(e ports.Share) bool { return samePath(e, s) })
	e := s
	e.Target = "http://127.0.0.1:" + strconv.Itoa(s.Reserved)
	f.entries = append(f.entries, e)
	if f.funnel == nil {
		f.funnel = map[int]bool{}
	}
	if s.Public() {
		f.funnel[s.Port] = true
	} else {
		delete(f.funnel, s.Port)
	}
	return "", nil
}

func (f *keptTailnet) Unshare(_ context.Context, s ports.Share) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, said("unshare", s))
	f.entries = slices.DeleteFunc(f.entries, func(e ports.Share) bool { return samePath(e, s) })
	if !slices.ContainsFunc(f.entries, func(e ports.Share) bool { return e.Port == s.Port }) {
		delete(f.funnel, s.Port)
	}
	return nil
}

func (f *keptTailnet) Shared(context.Context) ([]ports.Share, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := slices.Clone(f.entries)
	for i := range out {
		out[i].To = ""
		if !out[i].Mounted() && f.funnel[out[i].Port] {
			out[i].To = catalog.AudiencePublic
		}
	}
	return out, nil
}

func (f *keptTailnet) Funneled(context.Context) ([]int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var open []int
	for p, on := range f.funnel {
		if on {
			open = append(open, p)
		}
	}
	return open, nil
}

func (f *keptTailnet) dump() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.entries))
	for _, e := range f.entries {
		out = append(out, strconv.Itoa(e.Port)+e.Path+"<-"+strconv.Itoa(e.Reserved)+" funnel="+strconv.FormatBool(f.funnel[e.Port]))
	}
	slices.Sort(out)
	return strings.Join(out, "; ")
}

func funneledFrom40000() *keptTailnet {
	tn := &keptTailnet{fakeTailnet: fakeTailnet{sharing: ports.SharingState{On: true, DNSName: "x.tail1234.ts.net", Backend: ports.Running}}}
	tn.entries = []ports.Share{{Port: 443, Path: "/", Reserved: 40000, Scheme: ports.SchemeHTTPS, Target: "http://127.0.0.1:40000", To: catalog.AudiencePublic}}
	tn.funnel = map[int]bool{443: true}
	return tn
}

// invariant: a public share's Funnel entry the tailnet holds when the
// engine starts is station's: stopping the share removes it, so the next
// share given its port of the Mac is not on the internet.
func TestStopSharingAfterARestartRemovesItsFunnelEntry(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent})
	tn := funneledFrom40000()
	h.d.tn = tn
	h.d.o.Rights = func(context.Context, string) (*catalog.PortRights, error) {
		return &catalog.PortRights{Open: catalog.PortSet{{From: 3000, To: 3000}}, To: catalog.AudiencePublic}, nil
	}
	ctx := t.Context()
	yes := true
	h.d.reservations = []ports.Reservation{{VM: "api", GuestPort: 3000, MacPort: 40000, To: catalog.AudiencePublic, PublicPort: 443, HTTP: &yes}}
	h.d.refreshSelf(ctx)
	h.d.reconcileServe(ctx)
	if _, err := h.d.release(ctx, "api", 3000); err != nil {
		t.Fatal(err)
	}
	if got := tn.dump(); got != "" {
		t.Errorf("tailnet after stop-sharing = %s, want no entry", got)
	}
}

// invariant: rights narrowed from public to private, read at the VM's next
// run after an engine restart, unfunnel the port the tailnet already held.
func TestARightNarrowedAfterARestartUnfunnelsItsPort(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent})
	tn := funneledFrom40000()
	h.d.tn = tn
	rights := &catalog.PortRights{Open: catalog.PortSet{{From: 3000, To: 3000}}, To: catalog.AudiencePublic}
	h.d.o.Rights = func(context.Context, string) (*catalog.PortRights, error) { return rights, nil }
	ctx := t.Context()
	yes := true
	h.d.reservations = []ports.Reservation{{VM: "api", GuestPort: 3000, MacPort: 40000, To: catalog.AudiencePublic, PublicPort: 443, HTTP: &yes}}
	h.d.refreshSelf(ctx)
	h.d.reconcileServe(ctx)
	rights = &catalog.PortRights{Open: catalog.PortSet{{From: 3000, To: 3999}}, To: catalog.AudiencePrivate}
	h.d.loadRights(ctx, "api")
	h.d.mu.Lock()
	s := h.d.svcOf("api", 3000)
	s.st.Phase = ports.PhaseShared
	h.d.share(ctx, s)
	h.d.mu.Unlock()
	if got, want := tn.dump(), "40000<-40000 funnel=false"; got != want {
		t.Errorf("tailnet = %s, want %s", got, want)
	}
}

// invariant: an HTTPS mode switched while Serve is off is judged when Serve
// turns on, before any share is served: path mode never funnels 443.
func TestAModeSwitchedWhileServeIsOffIsJudgedWhenServeTurnsOn(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent})
	pol := &heldPolicy{g: &ports.PortsGroup{HTTPS: "port"}}
	h.d.st.Ports = pol
	tn := &fakeTailnet{sharing: ports.SharingState{On: true, DNSName: "x.tail1234.ts.net", Backend: ports.Running}}
	h.d.tn = tn
	ctx := t.Context()
	yes := true
	h.d.refreshSelf(ctx)
	h.d.mu.Lock()
	h.d.rights["api"] = ports.Rights{PortRights: catalog.PortRights{Open: catalog.PortSet{{From: 3000, To: 3000}}, To: catalog.AudiencePublic}}
	h.d.reservations = []ports.Reservation{{VM: "api", GuestPort: 3000, MacPort: 40000, To: catalog.AudiencePublic, PublicPort: 443, HTTP: &yes}}
	s := h.d.svcOf("api", 3000)
	s.st.Phase = ports.PhaseShared
	h.d.share(ctx, s)
	h.d.mu.Unlock()
	h.d.tickShares(ctx)
	tn.turn(false)
	h.d.refreshSelf(ctx)
	pol.g = &ports.PortsGroup{HTTPS: "path"}
	h.d.tickShares(ctx)
	tn.taken()
	tn.turn(true)
	h.d.refreshSelf(ctx)
	h.d.tickShares(ctx)
	calls := tn.taken()
	for _, c := range calls {
		if strings.HasPrefix(c, "share https 443") {
			t.Errorf("path mode funneled 443: %q", c)
		}
	}
	if !slices.Contains(calls, "share https 8443 from 40000") {
		t.Errorf("calls = %q, want 3000 shared on 8443", calls)
	}
	if got := h.d.reservations[0].PublicPort; got != 8443 {
		t.Errorf("public port = %d, want 8443", got)
	}
}

// invariant: a switch from path to port mode frees station's own private
// paths of 443 before a public share moves onto it.
func TestAPathToPortSwitchMovesThePublicShareOnto443(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent})
	pol := &heldPolicy{g: &ports.PortsGroup{HTTPS: "path"}}
	h.d.st.Ports = pol
	tn := &keptTailnet{fakeTailnet: fakeTailnet{sharing: ports.SharingState{On: true, DNSName: "x.tail1234.ts.net", Backend: ports.Running}}}
	h.d.tn = tn
	ctx := t.Context()
	yes := true
	h.d.refreshSelf(ctx)
	h.d.mu.Lock()
	h.d.rights["api"] = ports.Rights{PortRights: catalog.PortRights{Open: catalog.PortSet{{From: 3000, To: 3000}, {From: 3001, To: 3001}}, To: catalog.AudiencePublic}}
	h.d.reservations = []ports.Reservation{
		{VM: "api", GuestPort: 3000, MacPort: 40000, To: catalog.AudiencePublic, PublicPort: 8443, HTTP: &yes},
		{VM: "api", GuestPort: 5173, MacPort: 40001, By: ports.ByOwner, HTTP: &yes},
	}
	for _, p := range []int{3000, 5173} {
		s := h.d.svcOf("api", p)
		s.st.Phase = ports.PhaseShared
		h.d.share(ctx, s)
	}
	h.d.mu.Unlock()
	h.d.tickShares(ctx)
	pol.g = &ports.PortsGroup{HTTPS: "port"}
	h.d.tickShares(ctx)
	if got, want := tn.dump(), "40001<-40001 funnel=false; 443<-40000 funnel=true"; got != want {
		t.Errorf("tailnet = %s, want %s", got, want)
	}
	h.d.mu.Lock()
	defer h.d.mu.Unlock()
	if s := h.d.svcs[key("api", 3000)]; !s.served || s.serveNote != "" {
		t.Errorf("3000 served=%v note %q, want served on 443", s.served, s.serveNote)
	}
}

// invariant: a Funnel port held by an entry station made for another
// share says so, never that station did not make it.
func TestAFunnelPortHeldByStationsOwnEntryNamesIt(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent})
	tn := &keptTailnet{fakeTailnet: fakeTailnet{sharing: ports.SharingState{On: true, DNSName: "x.tail1234.ts.net", Backend: ports.Running}}}
	tn.entries = []ports.Share{{Port: 443, Path: "/", Reserved: 40001, Scheme: ports.SchemeHTTPS, Target: "http://127.0.0.1:40001"}}
	tn.funnel = map[int]bool{443: true}
	h.d.tn = tn
	err := h.d.heldOutside(t.Context(), ports.Share{Port: 443, Path: "/", Reserved: 40000, Scheme: ports.SchemeHTTPS, To: catalog.AudiencePublic})
	if want := "Funnel port 443 still serves station's entry from port 40001 of this Mac"; err == nil || err.Error() != want {
		t.Errorf("held = %v, want %q", err, want)
	}
}

// invariant: a private share on a port of the Mac is refused while that
// port is funneled by an entry station did not make: Funnel would put it
// on the internet.
func TestAPrivateShareOnAFunneledPortIsRefused(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent})
	tn := &keptTailnet{fakeTailnet: fakeTailnet{sharing: ports.SharingState{On: true, DNSName: "x.tail1234.ts.net", Backend: ports.Running}}}
	tn.funnel = map[int]bool{40001: true}
	h.d.tn = tn
	err := h.d.heldOutside(t.Context(), ports.Share{Port: 40001, Reserved: 40001, Scheme: ports.SchemeHTTPS})
	if want := "port 40001 is funneled by an entry station did not make: a private share on it would be public"; err == nil || err.Error() != want {
		t.Errorf("held = %v, want %q", err, want)
	}
}

// value: a kept tailnet that already holds the entry l makes in this
// HTTPS mode, as an engine restarted finds it.
func holding(l ports.Reservation, mode string) *keptTailnet {
	tn := &keptTailnet{fakeTailnet: fakeTailnet{sharing: ports.SharingState{On: true, DNSName: "x.tail1234.ts.net", Backend: ports.Running}}}
	e := ports.ShareOf(l, mode)
	e.Target = "http://127.0.0.1:" + strconv.Itoa(l.MacPort)
	tn.entries = []ports.Share{e}
	tn.funnel = map[int]bool{}
	if e.Public() {
		tn.funnel[e.Port] = true
	}
	return tn
}

// invariant: every path that drops a reservation unshares the entry it
// made while Serve is on, in the mode it was served in, whether this
// engine served it or found it at start.
func TestEveryPathThatDropsAReservationUnsharesItsEntry(t *testing.T) {
	yes := true
	public := catalog.PortRights{Open: catalog.PortSet{{From: 3000, To: 3000}, {From: 5000, To: 5000}, {From: 8080, To: 8080}}, To: catalog.AudiencePublic}
	private := catalog.PortRights{Open: catalog.PortSet{{From: 5173, To: 5173}}, To: catalog.AudiencePrivate}
	for _, c := range []struct {
		name   string
		mode   string
		rights catalog.PortRights
		l      ports.Reservation
		drop   func(context.Context, harness, *heldPolicy, *catalog.PortRights)
	}{
		{"a release", ports.HTTPSPort, private, ports.Reservation{VM: "api", GuestPort: 5173, MacPort: 40000, By: ports.ByRights, HTTP: &yes},
			func(ctx context.Context, h harness, _ *heldPolicy, _ *catalog.PortRights) {
				if _, err := h.d.release(ctx, "api", 5173); err != nil {
					t.Fatal(err)
				}
			}},
		{"an idle release", ports.HTTPSPort, public, ports.Reservation{VM: "api", GuestPort: 3000, MacPort: 40000, To: catalog.AudiencePublic, PublicPort: 443, By: ports.ByRights, HTTP: &yes},
			func(ctx context.Context, h harness, _ *heldPolicy, _ *catalog.PortRights) {
				h.d.mu.Lock()
				old := h.clock.At.Add(-8 * 24 * time.Hour)
				h.d.reservations[0].ReservedAt, h.d.reservations[0].LastSeenAt = old, old
				h.d.mu.Unlock()
				h.d.tickShares(ctx)
			}},
		{"a narrowing", ports.HTTPSPath, private, ports.Reservation{VM: "api", GuestPort: 5173, MacPort: 40000, By: ports.ByRights, HTTP: &yes},
			func(ctx context.Context, h harness, _ *heldPolicy, r *catalog.PortRights) {
				*r = catalog.PortRights{}
				h.d.loadRights(ctx, "api")
			}},
		{"a VM's delete", ports.HTTPSPort, public, ports.Reservation{VM: "api", GuestPort: 3000, MacPort: 40000, To: catalog.AudiencePublic, PublicPort: 443, By: ports.ByRights, HTTP: &yes},
			func(ctx context.Context, h harness, _ *heldPolicy, _ *catalog.PortRights) {
				if err := h.d.opDelete(ctx, "api"); err != nil {
					t.Fatal(err)
				}
			}},
		{"a mode switch", ports.HTTPSPort, public, ports.Reservation{VM: "api", GuestPort: 8080, MacPort: 40000, To: catalog.AudiencePublic, PublicPort: 10000, By: ports.ByRights, HTTP: &yes},
			func(ctx context.Context, h harness, pol *heldPolicy, _ *catalog.PortRights) {
				pol.g = &ports.PortsGroup{HTTPS: ports.HTTPSPath}
				h.d.tickShares(ctx)
			}},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent})
			pol := &heldPolicy{g: &ports.PortsGroup{HTTPS: c.mode}}
			h.d.st.Ports = pol
			tn := holding(c.l, c.mode)
			h.d.tn = tn
			rights := c.rights
			h.d.o.Rights = func(context.Context, string) (*catalog.PortRights, error) { return &rights, nil }
			ctx := t.Context()
			h.d.refreshSelf(ctx)
			h.d.adopt(ctx)
			h.d.mu.Lock()
			h.d.servedMode = c.mode
			h.d.reservations = []ports.Reservation{c.l}
			h.d.mu.Unlock()
			h.d.loadRights(ctx, "api")
			c.drop(ctx, h, pol, &rights)
			h.d.mu.Lock()
			left := len(h.d.reservations)
			h.d.mu.Unlock()
			if left != 0 {
				t.Fatalf("%d reservation(s) left, want the drop", left)
			}
			if got := tn.dump(); got != "" {
				t.Errorf("tailnet after %s = %s, want no entry", c.name, got)
			}
		})
	}
}

// invariant: an engine restarted after a public share's reservation
// passed its idle days unfunnels it at its first tick, so the next
// reservation given its port of the Mac is not on the internet.
func TestAnIdleReleaseAtTheFirstTickAfterARestartUnfunnelsItsPort(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent}, model.VMRecord{Name: "web", Lifetime: model.Persistent})
	tn := funneledFrom40000()
	h.d.tn = tn
	h.d.o.Rights = func(_ context.Context, vm string) (*catalog.PortRights, error) {
		if vm == "web" {
			return &catalog.PortRights{Open: catalog.PortSet{{From: 5432, To: 5432}}, To: catalog.AudienceLocal}, nil
		}
		return &catalog.PortRights{Open: catalog.PortSet{{From: 3000, To: 3000}}, To: catalog.AudiencePublic}, nil
	}
	ctx := t.Context()
	yes := true
	old := h.clock.At.Add(-8 * 24 * time.Hour)
	h.d.reservations = []ports.Reservation{{VM: "api", GuestPort: 3000, MacPort: 40000, To: catalog.AudiencePublic, PublicPort: 443, By: ports.ByRights, HTTP: &yes, ReservedAt: old, LastSeenAt: old}}
	h.d.refreshSelf(ctx)
	h.d.adopt(ctx)
	h.d.reconcileServe(ctx)
	h.d.tickShares(ctx)
	h.d.loadRights(ctx, "web")
	h.d.mu.Lock()
	w := h.d.svcOf("web", 5432)
	w.st.Phase = ports.PhaseShared
	h.d.share(ctx, w)
	h.d.mu.Unlock()
	if got := tn.dump(); strings.Contains(got, "<-40000") {
		t.Errorf("tailnet = %s, want no entry from 40000, now web:5432's (to: local)", got)
	}
}

// invariant: a release between an HTTPS mode switch and the tick that
// judges it unshares the entry in the mode it was served in.
func TestAReleaseBeforeTheTickOfAModeSwitchUnsharesTheModeItWasServedIn(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent})
	pol := &heldPolicy{g: &ports.PortsGroup{HTTPS: ports.HTTPSPort}}
	h.d.st.Ports = pol
	tn := &keptTailnet{fakeTailnet: fakeTailnet{sharing: ports.SharingState{On: true, DNSName: "x.tail1234.ts.net", Backend: ports.Running}}}
	h.d.tn = tn
	ctx := t.Context()
	yes := true
	h.d.refreshSelf(ctx)
	h.d.mu.Lock()
	h.d.servedMode = ports.HTTPSPort
	h.d.rights["api"] = ports.Rights{PortRights: catalog.PortRights{Open: catalog.PortSet{{From: 5173, To: 5173}}, To: catalog.AudiencePrivate}}
	h.d.reservations = []ports.Reservation{{VM: "api", GuestPort: 5173, MacPort: 40000, By: ports.ByRights, HTTP: &yes}}
	s := h.d.svcOf("api", 5173)
	s.st.Phase = ports.PhaseShared
	h.d.share(ctx, s)
	h.d.mu.Unlock()
	pol.g = &ports.PortsGroup{HTTPS: ports.HTTPSPath}
	if _, err := h.d.release(ctx, "api", 5173); err != nil {
		t.Fatal(err)
	}
	if got := tn.dump(); got != "" {
		t.Errorf("tailnet after the release = %s, want no entry", got)
	}
}

// invariant: a path entry carried from the range is station's whether or
// not its VM is still known: a VM deleted while Serve is off leaves its
// path entry, and Serve turning on removes it before the next reservation
// on its port of the Mac rides it.
func TestAPathEntryOfADeletedVMGoesWhenServeTurnsOn(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent})
	h.d.st.Ports = &heldPolicy{g: &ports.PortsGroup{HTTPS: ports.HTTPSPath}}
	yes := true
	l := ports.Reservation{VM: "api", GuestPort: 5173, MacPort: 40000, By: ports.ByRights, HTTP: &yes}
	tn := holding(l, ports.HTTPSPath)
	h.d.tn = tn
	ctx := t.Context()
	tn.turn(false)
	h.d.refreshSelf(ctx)
	h.d.adopt(ctx)
	h.d.mu.Lock()
	h.d.reservations = []ports.Reservation{l}
	h.d.mu.Unlock()
	if err := h.d.opDelete(ctx, "api"); err != nil {
		t.Fatal(err)
	}
	tn.turn(true)
	h.d.refreshSelf(ctx)
	if got := tn.dump(); got != "" {
		t.Errorf("tailnet after Serve on = %s, want no entry", got)
	}
}

// invariant: what a VM at its most refuses is told as one count, and goes
// with the listeners: a thousand ports that come and go leave none.
func TestAFullVMsRefusedListenersLeaveNothingWhenTheyGo(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent})
	h.d.st.Ports = &heldPolicy{g: &ports.PortsGroup{MaxPerVM: 1}}
	h.d.tn = &fakeTailnet{sharing: ports.SharingState{On: true, DNSName: "x.tail1234.ts.net", Backend: ports.Running}}
	h.d.o.Rights = func(context.Context, string) (*catalog.PortRights, error) {
		return &catalog.PortRights{Open: catalog.PortSet{{From: 9000, To: 9999}}, To: catalog.AudiencePrivate}, nil
	}
	ctx := t.Context()
	h.d.refreshSelf(ctx)
	h.d.adopt(ctx)
	h.d.mu.Lock()
	h.d.vms["api"].state = engine.StateRunning
	h.d.mu.Unlock()
	h.d.listen(ctx, "api", 9000, "0.0.0.0", "node")
	for port := 9001; port < 9101; port++ {
		h.d.listen(ctx, "api", port, "0.0.0.0", "node")
	}
	got := h.d.macPorts("").NotShared
	if len(got) != 1 || got[0].VM != "api" || got[0].Count != 100 || got[0].Detail == "" {
		t.Fatalf("not shared = %+v, want api's 100 as one line", got)
	}
	for port := 9001; port < 9101; port++ {
		h.d.unlisten("api", port)
	}
	if got := h.d.macPorts("").NotShared; len(got) != 0 {
		t.Errorf("not shared = %+v after the listeners went, want none", got)
	}
	h.d.mu.Lock()
	defer h.d.mu.Unlock()
	if len(h.d.svcs) != 1 {
		t.Errorf("%d services kept, want the one shared", len(h.d.svcs))
	}
}

// invariant: a listener its guest marks quiet is never heard nor shared by
// itself, and the owner's reservation shares it at the next tick.
func TestAQuietListenerIsSharedOnlyOnceTheOwnerReservesIt(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent})
	h.d.tn = &fakeTailnet{sharing: ports.SharingState{On: true, DNSName: "x.tail1234.ts.net", Backend: ports.Running}}
	h.d.o.Rights = func(context.Context, string) (*catalog.PortRights, error) {
		return &catalog.PortRights{Open: catalog.PortSet{{From: 1024, To: 65535}}, To: catalog.AudiencePrivate}, nil
	}
	ctx := t.Context()
	h.d.refreshSelf(ctx)
	h.d.adopt(ctx)
	h.d.mu.Lock()
	h.d.vms["api"].state = engine.StateRunning
	h.d.mu.Unlock()
	snapshot := `{"type":"snapshot","ports":[{"port":3000,"ip":"0.0.0.0","process":"node"},{"port":5173,"ip":"127.0.0.1","process":"vite"},{"port":45000,"ip":"0.0.0.0","process":"node","quiet":true}]}`
	h.d.fromGuest(ctx, h.d.vms["api"], []byte(snapshot))
	if heard := h.d.heardOf("api"); len(heard) != 2 {
		t.Fatalf("heard %+v, want the plain and the loopback listener, not the quiet one", heard)
	}
	h.d.mu.Lock()
	shared := len(h.d.svcs)
	h.d.mu.Unlock()
	if shared != 2 {
		t.Fatalf("%d services, want the plain and the loopback listener's", shared)
	}
	if _, err := h.d.guestRequest(ctx, "api", guestMessage{Op: "reserve", Port: 45000}); err != nil {
		t.Fatal(err)
	}
	h.d.tickShares(ctx)
	h.d.mu.Lock()
	defer h.d.mu.Unlock()
	if s := h.d.svcs[key("api", 45000)]; s == nil || s.st.Phase != ports.PhaseShared {
		t.Errorf("port 45000 service = %+v, want shared once reserved", s)
	}
}
