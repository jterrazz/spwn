package process

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	model "github.com/jterrazz/studio-station/src/machines"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/ports"
)

type fakeTailnet struct {
	mu      sync.Mutex
	sharing ports.SharingState
	shared  []ports.Share
	calls   []string
	certs   []string
	hold    chan struct{}
	certErr error
	until   time.Time
	entered chan string
}

func (f *fakeTailnet) Warm(ctx context.Context) error {
	f.mu.Lock()
	mac := f.sharing.DNSName
	f.certs = append(f.certs, mac)
	hold, err := f.hold, f.certErr
	f.until, _ = ctx.Deadline()
	f.mu.Unlock()
	if f.entered != nil {
		f.entered <- mac
	}
	if hold != nil {
		<-hold
	}
	return err
}

func (f *fakeTailnet) issued() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.certs...)
}

func (f *fakeTailnet) Ready(context.Context) (ports.SharingState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sharing, nil
}

func said(verb string, s ports.Share) string {
	return verb + " " + s.Scheme + " " + strconv.Itoa(s.Port) + s.Path + " from " + strconv.Itoa(s.Reserved)
}

func (f *fakeTailnet) Share(_ context.Context, s ports.Share) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, said("share", s))
	return "", nil
}

func (f *fakeTailnet) Unshare(_ context.Context, s ports.Share) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, said("unshare", s))
	return nil
}

func (f *fakeTailnet) Shared(context.Context) ([]ports.Share, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "shared")
	return f.shared, nil
}

func (f *fakeTailnet) turn(on bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sharing.On = on
}

func (f *fakeTailnet) taken() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.calls
	f.calls = nil
	return out
}

func TestServeIsReadEveryThirtyTicksAndOnlyServesWhileOn(t *testing.T) {
	h := newHarness(t)
	tn := &fakeTailnet{
		sharing: ports.SharingState{DNSName: "x.tail1234.ts.net", Backend: ports.Running},
		shared:  []ports.Share{{Port: 40099, Scheme: ports.SchemeTCP, Target: "127.0.0.1:40099"}},
	}
	h.d.tn = tn
	ready := certLines(h.d)
	ctx := t.Context()
	yes := true
	h.d.reservations = []ports.Reservation{{VM: "api", GuestPort: 8000, MacPort: 40000, HTTP: &yes}}
	h.d.refreshSelf(ctx)
	h.d.mu.Lock()
	s := h.d.svcOf("api", 8000)
	s.st.Phase = ports.PhaseShared
	h.d.share(ctx, s)
	h.d.mu.Unlock()
	defer h.d.closeAll()
	row := func() ports.SharedPort { return h.d.macPorts("").Shared[0] }
	if calls := tn.taken(); len(calls) != 0 || row().Detail != catalog.ServeOffDetail || row().URLs.Public != "http://127.0.0.1:40000" {
		t.Fatalf("Serve off: calls %q, row %+v; want no serve, the admin step and the local URL", calls, row())
	}
	tn.turn(true)
	for tick := 1; tick < serveEvery; tick++ {
		h.d.step(ctx, tick)
	}
	if calls := tn.taken(); len(calls) != 0 {
		t.Fatalf("Serve turned on was acted on before its read: %q", calls)
	}
	h.d.step(ctx, serveEvery)
	<-ready
	want := []string{"share https 40000 from 40000", "shared", "unshare tcp 40099 from 0"}
	if calls := tn.taken(); strings.Join(calls, "|") != strings.Join(want, "|") {
		t.Fatalf("Serve on at the next read: calls %q, want %q", calls, want)
	}
	if r := row(); r.Detail != "" || r.URLs.Public != "https://x.tail1234.ts.net:40000" {
		t.Fatalf("Serve on: row %+v, want a clean https publish", r)
	}
	tn.turn(false)
	for tick := serveEvery + 1; tick <= 2*serveEvery; tick++ {
		h.d.step(ctx, tick)
	}
	h.d.mu.Lock()
	h.d.unpublish(ctx, s)
	h.d.mu.Unlock()
	if calls := tn.taken(); len(calls) != 0 {
		t.Errorf("Serve off again: calls %q, want no serve at all", calls)
	}
	if r := row(); r.Detail != catalog.ServeOffDetail || r.URLs.Public != "http://127.0.0.1:40000" {
		t.Errorf("Serve off again: row %+v, want the admin step and the local URL", r)
	}
}

func TestARunRendersTheProfileEnvironmentAndAnAdoptionKeepsIt(t *testing.T) {
	kept := map[string]string{"EXPO_PACKAGER_PROXY_URL": "http://127.0.0.1:40000"}
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent, Want: model.WantRunning, Env: kept})
	h.d.tn = &fakeTailnet{sharing: ports.SharingState{DNSName: "x.tail1234.ts.net", Backend: ports.Running, On: true}}
	yes := true
	expo := catalog.Profile{Name: "expo", Port: 8081, HTTP: true, Env: []catalog.ProfileEnv{{Name: "EXPO_PACKAGER_PROXY_URL", Value: "{url}"}}}
	h.d.o.Profiles = []catalog.Profile{expo}
	h.d.reservations = []ports.Reservation{{VM: "api", GuestPort: expo.Port, MacPort: 40000, HTTP: &yes, Pinned: true, Profile: expo.Name}}
	h.d.refreshSelf(t.Context())
	h.hv.outlived("api")
	ctx := cancelled(t)
	h.d.adopt(ctx)
	if len(h.m.saved) != 0 {
		t.Fatalf("an adoption rewrote the record: %+v", h.m.saved)
	}
	v, _ := h.d.find("api")
	h.hv.end("api", model.RunEnd{Reason: model.GuestReboot})
	_ = h.d.start(ctx, v)
	if len(h.m.saved) != 1 || h.m.saved[0].Env["EXPO_PACKAGER_PROXY_URL"] != "https://x.tail1234.ts.net:40000" || h.m.saved[0].Env["STATION_PUBLIC_HOST"] != "x.tail1234.ts.net" {
		t.Errorf("a run with Serve on saved %+v, want the https URL and the public host", h.m.saved)
	}
}

func certLines(d *Engine) chan string {
	lines := make(chan string, 8)
	d.note = func(line string) {
		if strings.Contains(line, "certificate for") {
			lines <- line
		}
	}
	return lines
}

func TestTheCertificateIsWarmedWhenServeTurnsOnOneAtATime(t *testing.T) {
	h := newHarness(t)
	tn := &fakeTailnet{
		sharing: ports.SharingState{DNSName: "x.tail1234.ts.net", Backend: ports.Running, On: true},
		hold:    make(chan struct{}), entered: make(chan string, 4),
	}
	h.d.tn = tn
	ready := certLines(h.d)
	ctx := t.Context()
	yes := true
	h.d.reservations = []ports.Reservation{{VM: "api", GuestPort: 8000, MacPort: 40000, HTTP: &yes}}
	h.d.refreshSelf(ctx)
	<-tn.entered
	h.d.mu.Lock()
	s := h.d.svcOf("api", 8000)
	s.st.Phase = ports.PhaseShared
	h.d.share(ctx, s)
	h.d.mu.Unlock()
	defer h.d.closeAll()
	if r := h.d.macPorts("").Shared[0]; r.Detail != catalog.IssuingDetail || r.URLs.Public != "https://x.tail1234.ts.net:40000" {
		t.Fatalf("while the certificate is issued: row %+v, want the https URL with %q", r, catalog.IssuingDetail)
	}
	tn.turn(false)
	h.d.step(ctx, serveEvery)
	tn.turn(true)
	h.d.step(ctx, 2*serveEvery)
	if got := tn.issued(); len(got) != 1 || got[0] != "x.tail1234.ts.net" {
		t.Fatalf("Serve on at start, then off and on during the warm-up: certs %q, want one for the host", got)
	}
	close(tn.hold)
	if line := <-ready; !strings.HasSuffix(line, "tailnet: certificate for x.tail1234.ts.net ready in 0s") {
		t.Errorf("daemon.log %q, want the ready line", line)
	}
	for tick := 2*serveEvery + 1; tick <= 3*serveEvery; tick++ {
		h.d.step(ctx, tick)
	}
	if got := tn.issued(); len(got) != 1 {
		t.Fatalf("Serve still on: certs %q, want no second warm-up", got)
	}
	if r := h.d.macPorts("").Shared[0]; r.Detail != "" {
		t.Fatalf("the certificate ready: row detail %q, want none", r.Detail)
	}
	tn.turn(false)
	h.d.step(ctx, 4*serveEvery)
	tn.turn(true)
	h.d.step(ctx, 5*serveEvery)
	<-tn.entered
	<-ready
	if got := tn.issued(); len(got) != 2 {
		t.Fatalf("Serve off, then on again: certs %q, want a second warm-up", got)
	}
}

func TestACertificateNeverIssuedIsCutAtItsBudgetAndSaysSo(t *testing.T) {
	h := newHarness(t)
	tn := &fakeTailnet{sharing: ports.SharingState{DNSName: "x.tail1234.ts.net", Backend: ports.Running}, certErr: context.DeadlineExceeded}
	h.d.tn = tn
	var lines []string
	done := make(chan struct{})
	h.d.note = func(line string) {
		if strings.Contains(line, "certificate for") {
			lines = append(lines, line)
			close(done)
		}
	}
	h.d.refreshSelf(t.Context())
	if got := tn.issued(); len(got) != 0 {
		t.Fatalf("Serve off at start: certs %q, want none", got)
	}
	tn.turn(true)
	h.d.step(t.Context(), serveEvery)
	<-done
	tn.mu.Lock()
	until := tn.until
	tn.mu.Unlock()
	if want := h.clock.At.Add(certBudget); !until.Equal(want) {
		t.Errorf("the warm-up's deadline: %s, want %s", until, want)
	}
	if len(lines) != 1 || !strings.HasSuffix(lines[0], "tailnet: certificate for x.tail1234.ts.net not issued: no answer within 2m0s") {
		t.Errorf("daemon.log %q, want the not-issued line", lines)
	}
}

func TestAnEngineThatStopsLeavesItsRunningVMsPortsSharedOnTheTailnet(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent, Want: model.WantRunning})
	h.hv.outlived("api")
	tn := &fakeTailnet{sharing: ports.SharingState{On: true, DNSName: "x.tail1234.ts.net", Backend: ports.Running}}
	h.d.tn = tn
	ctx := t.Context()
	h.d.adopt(cancelled(t))
	yes := true
	h.d.reservations = []ports.Reservation{{VM: "api", GuestPort: 5173, MacPort: 40001, HTTP: &yes, Pinned: true}}
	h.d.refreshSelf(ctx)
	h.d.mu.Lock()
	s := h.d.svcOf("api", 5173)
	s.st.Phase = ports.PhaseShared
	h.d.share(ctx, s)
	h.d.mu.Unlock()
	tn.taken()
	v, err := h.d.find("api")
	if err != nil {
		t.Fatal(err)
	}
	h.d.mu.Lock()
	gen := v.gen
	h.d.mu.Unlock()
	h.d.session(cancelled(t), v, gen)
	for _, call := range tn.taken() {
		if strings.HasPrefix(call, "unshare") {
			t.Fatalf("the engine stopping took back %q: api still runs, so its port stays shared", call)
		}
	}
}
