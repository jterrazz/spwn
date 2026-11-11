package process

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/ports"
)

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
	var out []string
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
