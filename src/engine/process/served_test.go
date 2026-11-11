package process

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/engine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/ports"
)

// value: a Mac whose loopback ports in busy are held by another of its
// processes.
type busyMac struct {
	*fakeMac
	busy map[int]bool
}

func (b *busyMac) ListenLoopback(p int) (Listener, error) {
	if b.busy[p] {
		return nil, errors.New("bind: address already in use")
	}
	return b.fakeMac.ListenLoopback(p)
}

// value: a guest whose every port is a server that speaks first, as ssh
// does: not HTTP.
type speaksFirst struct{ done chan struct{} }

func (speaksFirst) Exec(context.Context, string, string) (engine.GuestRan, error) {
	return engine.GuestRan{}, errors.New("no exec in a module test")
}

func (speaksFirst) Attach(context.Context, string) (Stream, error) {
	return nil, errors.New("no session in a module test")
}

func (speaksFirst) Forward(context.Context, string, int) (Stream, error) {
	return &banner{left: []byte("SSH-2.0-OpenSSH_9.9\r\n")}, nil
}

// value: a stream that says its banner, then ends.
type banner struct{ left []byte }

func (b *banner) Read(p []byte) (int, error) {
	if len(b.left) == 0 {
		return 0, errors.New("closed")
	}
	n := copy(p, b.left)
	b.left = b.left[n:]
	return n, nil
}

func (*banner) Write(p []byte) (int, error) { return len(p), nil }
func (*banner) Close() error                { return nil }

func (speaksFirst) Close() error            { return nil }
func (g speaksFirst) Done() <-chan struct{} { return g.done }

func (f *keptTailnet) said() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls)
}

// value: an engine restarted over a tailnet that holds the entry l makes
// in mode, its VM api running, l's port of the Mac now free; the
// reservation is marked served, as a start finds it.
func restartedOver(t *testing.T, l ports.Reservation, mode string, rights *catalog.PortRights) (harness, *heldPolicy, *keptTailnet, *busyMac) {
	t.Helper()
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent})
	pol := &heldPolicy{g: &ports.PortsGroup{HTTPS: mode}}
	h.d.st.Ports = pol
	tn := holding(l, mode)
	h.d.tn = tn
	mac := &busyMac{fakeMac: h.mac, busy: map[int]bool{}}
	h.d.h = mac
	h.d.o.Rights = func(context.Context, string) (*catalog.PortRights, error) { return rights, nil }
	ctx := t.Context()
	h.d.reservations = []ports.Reservation{l}
	h.d.servedMode = mode
	h.d.refreshSelf(ctx)
	h.d.adopt(ctx)
	h.d.mu.Lock()
	v := h.d.vms["api"]
	if v == nil {
		h.d.mu.Unlock()
		t.Fatal("api not adopted")
	}
	v.state = engine.StateRunning
	h.d.mu.Unlock()
	h.d.reconcileServe(ctx)
	return h, pol, tn, mac
}

// invariant: every change to what a reservation serves (its port of the
// Mac, its scheme, its audience, its Funnel port, the HTTPS mode) unshares
// the entry it served before, then serves the new one; a change that
// leaves what it serves as it was unshares nothing.
func TestEveryChangeToWhatAReservationServesUnsharesTheOldEntryFirst(t *testing.T) {
	yes := true
	public := catalog.PortRights{Open: catalog.PortSet{{From: 3000, To: 3000}, {From: 5000, To: 5000}}, To: catalog.AudiencePublic}
	private := catalog.PortRights{Open: catalog.PortSet{{From: 3000, To: 3000}, {From: 5000, To: 5000}}, To: catalog.AudiencePrivate}
	local := catalog.PortRights{Open: catalog.PortSet{{From: 3000, To: 3000}}, To: catalog.AudienceLocal}
	privateAt := func(mac int) ports.Reservation {
		return ports.Reservation{VM: "api", GuestPort: 3000, MacPort: mac, By: ports.ByRights, HTTP: &yes}
	}
	publicAt := func(funnel int) ports.Reservation {
		return ports.Reservation{VM: "api", GuestPort: 3000, MacPort: 40000, To: catalog.AudiencePublic, PublicPort: funnel, By: ports.ByRights, HTTP: &yes}
	}
	listen := func(ctx context.Context, h harness) { h.d.listen(ctx, "api", 3000, "0.0.0.0", "node") }
	rightsTo := func(r catalog.PortRights) func(context.Context, harness, *heldPolicy, *catalog.PortRights, *busyMac) {
		return func(ctx context.Context, h harness, _ *heldPolicy, now *catalog.PortRights, _ *busyMac) {
			listen(ctx, h)
			*now = r
			h.d.loadRights(ctx, "api")
		}
	}
	modeTo := func(mode string) func(context.Context, harness, *heldPolicy, *catalog.PortRights, *busyMac) {
		return func(ctx context.Context, h harness, pol *heldPolicy, _ *catalog.PortRights, _ *busyMac) {
			listen(ctx, h)
			pol.g = &ports.PortsGroup{HTTPS: mode}
			h.d.tickShares(ctx)
		}
	}
	movedOffItsPort := func(ctx context.Context, h harness, _ *heldPolicy, _ *catalog.PortRights, mac *busyMac) {
		mac.busy[40000] = true
		listen(ctx, h)
	}
	for _, c := range []struct {
		name   string
		mode   string
		rights catalog.PortRights
		l      ports.Reservation
		change func(context.Context, harness, *heldPolicy, *catalog.PortRights, *busyMac)
		want   string
	}{
		{"a public share moved off a busy port of the Mac", ports.HTTPSPort, public, publicAt(443), movedOffItsPort, "443<-40001 funnel=true"},
		{"a private share moved off a busy port of the Mac", ports.HTTPSPort, private, privateAt(40000), movedOffItsPort, "40001<-40001 funnel=false"},
		{"a path share moved off a busy port of the Mac", ports.HTTPSPath, private, privateAt(40000), movedOffItsPort, "443/api/3000<-40001 funnel=false"},
		{"a scheme told by a server that is not HTTP", ports.HTTPSPort, private, ports.Reservation{VM: "api", GuestPort: 3000, MacPort: 40000, By: ports.ByRights},
			func(ctx context.Context, h harness, _ *heldPolicy, _ *catalog.PortRights, _ *busyMac) {
				h.d.mu.Lock()
				h.d.vms["api"].client = speaksFirst{done: make(chan struct{})}
				h.d.mu.Unlock()
				listen(ctx, h)
			}, "40000<-40000 funnel=false"},
		{"an audience widened to public", ports.HTTPSPort, private, privateAt(40000), rightsTo(public), "443<-40000 funnel=true"},
		{"an audience narrowed to private", ports.HTTPSPort, public, publicAt(443), rightsTo(private), "40000<-40000 funnel=false"},
		{"an audience narrowed to local", ports.HTTPSPort, private, privateAt(40000), rightsTo(local), ""},
		{"a Funnel port moved by the rights", ports.HTTPSPort, public, publicAt(443), rightsTo(catalog.PortRights{Open: catalog.PortSet{{From: 5000, To: 5000}, {From: 3000, To: 3000}}, To: catalog.AudiencePublic}), "8443<-40000 funnel=true"},
		{"a Funnel port moved by the HTTPS mode", ports.HTTPSPort, public, publicAt(443), modeTo(ports.HTTPSPath), "8443<-40000 funnel=true"},
		{"a private share switched from port to path", ports.HTTPSPort, private, privateAt(40000), modeTo(ports.HTTPSPath), "443/api/3000<-40000 funnel=false"},
		{"a private share switched from path to port", ports.HTTPSPath, private, privateAt(40000), modeTo(ports.HTTPSPort), "40000<-40000 funnel=false"},
		{"a private share switched from port to off", ports.HTTPSPort, private, privateAt(40000), modeTo(ports.HTTPSOff), "40000<-40000 funnel=false"},
	} {
		t.Run(c.name, func(t *testing.T) {
			rights := c.rights
			h, pol, tn, mac := restartedOver(t, c.l, c.mode, &rights)
			ctx := t.Context()
			c.change(ctx, h, pol, &rights, mac)
			if got := tn.dump(); got != c.want {
				t.Errorf("tailnet after %s = %q, want %q", c.name, got, c.want)
			}
			if old := said("unshare", ports.ShareOf(c.l, c.mode)); !slices.Contains(tn.said(), old) {
				t.Errorf("calls after %s = %v, want %q", c.name, tn.said(), old)
			}
		})
	}
	t.Run("a share the owner takes from its rights", func(t *testing.T) {
		private := private
		h, _, tn, _ := restartedOver(t, privateAt(40000), ports.HTTPSPort, &private)
		ctx := t.Context()
		listen(ctx, h)
		if _, err := h.d.reserve(ctx, "api", 3000, ports.ByOwner); err != nil {
			t.Fatal(err)
		}
		if got, want := tn.dump(), "40000<-40000 funnel=false"; got != want {
			t.Errorf("tailnet = %q, want %q", got, want)
		}
		if calls := strings.Join(tn.said(), "; "); strings.Contains(calls, "unshare") {
			t.Errorf("calls = %s, want no unshare", calls)
		}
	})
}

// invariant: a share moved off a port of the Mac another process took
// leaves nothing on the old port: its release then removes every entry,
// and the next reservation given that port, a local one, is on no
// tailnet.
func TestAShareMovedOffABusyPortLeavesNothingOnTheOldOne(t *testing.T) {
	yes := true
	rights := catalog.PortRights{Open: catalog.PortSet{{From: 3000, To: 3000}}, To: catalog.AudiencePrivate}
	h, _, tn, mac := restartedOver(t, ports.Reservation{VM: "api", GuestPort: 3000, MacPort: 40000, By: ports.ByRights, HTTP: &yes}, ports.HTTPSPort, &rights)
	ctx := t.Context()
	mac.busy[40000] = true
	h.d.listen(ctx, "api", 3000, "0.0.0.0", "node")
	if _, err := h.d.release(ctx, "api", 3000); err != nil {
		t.Fatal(err)
	}
	delete(mac.busy, 40000)
	h.d.mu.Lock()
	h.d.rights["web"] = ports.Rights{PortRights: catalog.PortRights{Open: catalog.PortSet{{From: 5432, To: 5432}}, To: catalog.AudienceLocal}}
	w := h.d.svcOf("web", 5432)
	w.st.Phase = ports.PhaseShared
	h.d.share(ctx, w)
	taken := h.d.reservations[0].MacPort
	h.d.mu.Unlock()
	if taken != 40000 {
		t.Fatalf("web:5432 took %d, want 40000", taken)
	}
	if got := tn.dump(); got != "" {
		t.Errorf("tailnet = %s, want no entry", got)
	}
}
