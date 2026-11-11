package process

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"

	"github.com/jterrazz/studio-station/src/engine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/ports"
)

const (
	bannerWait = 300 * time.Millisecond
	headWait   = 2 * time.Second
)

type svc struct {
	vm        string
	port      int
	ip        string
	process   string
	st        ports.State
	listeners []Listener
	served    bool
	detail    string
	serveNote string
	refused   bool
}

func key(vm string, port int) string { return vm + ":" + strconv.Itoa(port) }

func (d *Engine) reservationIndex(vm string, port int) int {
	_, i := ports.Find(d.reservations, vm, port)
	return i
}

func (d *Engine) busy(p int) bool {
	l, err := d.h.ListenLoopback(p)
	if err != nil {
		return true
	}
	_ = l.Close()
	return false
}

func asOp(err error) error {
	var full *ports.RangeFull
	var perVM *ports.VMFull
	switch {
	case errors.As(err, &full):
		return fault("range_full", err.Error(), "station ports stop-sharing <vm> <port>")
	case errors.As(err, &perVM):
		return perVM.Refusal()
	}
	return fault("internal", err.Error(), "station ports")
}

func (d *Engine) svcOf(vm string, port int) *svc {
	s, ok := d.svcs[key(vm, port)]
	if !ok {
		s = &svc{vm: vm, port: port}
		if i := d.reservationIndex(vm, port); i >= 0 {
			s.st.LastSeenAt = cmp.Or(d.reservations[i].LastSeenAt, d.reservations[i].ReservedAt)
			s.st.Phase = ports.PhaseWaiting
		}
		d.svcs[key(vm, port)] = s
	}
	return s
}

func (d *Engine) apply(ctx context.Context, s *svc, acts []ports.Action) {
	for _, a := range acts {
		switch a {
		case ports.ActShare:
			d.share(ctx, s)
		case ports.ActUnpublish:
			d.unpublish(ctx, s)
		case ports.ActRelease:
			d.dropLocked(ctx, s.vm, s.port)
			delete(d.svcs, key(s.vm, s.port))
		}
	}
}

// invariant: a port that starts listening inside its VM's rights is
// shared with their audience; one outside them is flagged, never shared,
// until the owner shares it. A server with no rights shares nothing by
// itself, its tooling choosing its routes, and flags nothing.
func (d *Engine) listen(ctx context.Context, vm string, port int, ip, process string) {
	r := d.rightsOf(ctx, vm)
	p := d.policy()
	if !r.Reported(port, d.o.GuestPort) || p.Stopped(vm, port) {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	reserved := d.reservationIndex(vm, port) >= 0
	if v := d.vms[vm]; v != nil && v.rec.Lifetime == model.AlwaysOn && !reserved && !r.Opens() {
		return
	}
	s := d.svcOf(vm, port)
	s.ip, s.process = ip, process
	if !reserved && !r.Allows(port) {
		s.st.Phase, s.detail = ports.PhaseNotAllowed, ports.NotAllowed(r)
		return
	}
	now := d.h.Now().UTC().Truncate(time.Second)
	if s.st.Phase == ports.PhaseRefused || s.st.Phase == ports.PhaseNotAllowed {
		s.st.Phase, s.detail = ports.PhaseNone, ""
	}
	var acts []ports.Action
	s.st, acts = ports.Next(s.st, ports.EvListen, now)
	if i := d.reservationIndex(vm, port); i >= 0 && now.Sub(d.reservations[i].LastSeenAt) >= seenEvery {
		d.reservations[i].LastSeenAt = now
		d.saveReservations()
	}
	d.apply(ctx, s, acts)
}

func (d *Engine) unlisten(vm string, port int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if s, ok := d.svcs[key(vm, port)]; ok {
		if s.st.Phase == ports.PhaseNotAllowed {
			delete(d.svcs, key(vm, port))
			return
		}
		s.st, _ = ports.Next(s.st, ports.EvUnlisten, d.h.Now())
	}
}

func (d *Engine) unlistenExcept(vm string, seen map[int]bool) {
	d.mu.Lock()
	var gone []int
	for _, s := range d.svcs {
		if s.vm == vm && !seen[s.port] {
			gone = append(gone, s.port)
		}
	}
	d.mu.Unlock()
	for _, port := range gone {
		d.unlisten(vm, port)
	}
}

func (d *Engine) tickShares(ctx context.Context) {
	// invariant: every reserved port ticks, listened on in this engine's
	// life or not, so one idle past its days is released, and one served
	// that its VM says nothing listens on is unshared.
	mode := d.policy().HTTPS
	d.mu.Lock()
	defer d.mu.Unlock()
	d.serveInModeLocked(ctx, mode)
	now := d.h.Now()
	for _, l := range slices.Clone(d.reservations) {
		d.svcOf(l.VM, l.GuestPort)
	}
	for _, s := range d.svcs {
		var acts []ports.Action
		s.st, acts = ports.Next(s.st, ports.EvTick, now)
		d.apply(ctx, s, acts)
		_, told := d.heard[s.vm]
		if v := d.vms[s.vm]; s.st.Phase == ports.PhaseWaiting && s.served && (told || v == nil || v.state != engine.StateRunning) {
			d.unpublish(ctx, s)
		}
	}
}

// invariant: HTTPS links turned on or off on this Mac serve every served
// port again in the new mode at the next tick, each unshared first in the
// mode it was served in; while Serve is off the mode last served in is
// kept, so a switch made then is judged when Serve turns on. The caller
// holds d.mu.
func (d *Engine) serveInModeLocked(ctx context.Context, mode string) {
	if !d.self.On {
		return
	}
	was := d.servedMode
	d.servedMode = mode
	if was == "" || was == mode {
		return
	}
	// invariant: the private shares leave the old mode, then the Funnel
	// ports move with the mode, then the private shares are served in it:
	// path mode frees 443 of every public share before a path takes it,
	// and port mode frees it of every path before a public share does.
	var private []*svc
	for _, s := range d.svcs {
		i := d.reservationIndex(s.vm, s.port)
		if i < 0 || !s.served || d.reservations[i].Audience() != catalog.AudiencePrivate {
			continue
		}
		d.unshareLocked(ctx, d.reservations[i], was, true)
		s.served = false
		private = append(private, s)
	}
	vms := map[string]bool{}
	for _, l := range d.reservations {
		vms[l.VM] = true
	}
	for vm := range vms {
		if r, ok := d.rights[vm]; ok {
			d.judgeLocked(ctx, vm, r, mode)
		}
	}
	for _, s := range private {
		i := d.reservationIndex(s.vm, s.port)
		if i < 0 || s.served || d.svcs[key(s.vm, s.port)] != s || d.reservations[i].Audience() != catalog.AudiencePrivate {
			continue
		}
		d.expose(ctx, s, d.reservations[i])
	}
}

func (d *Engine) vmStopped(ctx context.Context, vm string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if ls := d.heard[vm]; len(ls) > 0 {
		d.lastHeard[vm] = ls
	}
	delete(d.heard, vm)
	for _, s := range d.svcs {
		if s.vm == vm && s.st.Phase == ports.PhaseNotAllowed {
			delete(d.svcs, key(s.vm, s.port))
			continue
		}
		if s.vm == vm {
			var acts []ports.Action
			s.st, acts = ports.Next(s.st, ports.EvVMStopped, d.h.Now())
			d.apply(ctx, s, acts)
		}
	}
}

func (d *Engine) releaseVM(ctx context.Context, vm string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, l := range slices.Clone(d.reservations) {
		if l.VM == vm {
			d.dropLocked(ctx, l.VM, l.GuestPort)
		}
	}
	for _, s := range d.svcs {
		if s.vm == vm {
			d.unpublish(ctx, s)
			delete(d.svcs, key(s.vm, s.port))
		}
	}
}

// invariant: every path that drops a reservation comes through here: a
// release, an idle release, a narrowing, a VM's delete, a mode switch that
// leaves it no Funnel port. Its listeners close and its entry goes through
// unshareLocked. The caller holds d.mu.
func (d *Engine) dropLocked(ctx context.Context, vm string, port int) {
	i := d.reservationIndex(vm, port)
	if i < 0 {
		return
	}
	l := d.reservations[i]
	s := d.svcs[key(vm, port)]
	served := s != nil && s.served
	if s != nil {
		d.unpublish(ctx, s)
	}
	if !served {
		d.unshareLocked(ctx, l, d.servedModeLocked(), false)
	}
	d.reservations = append(d.reservations[:i:i], d.reservations[i+1:]...)
	d.saveReservations()
}

// invariant: every change to a kept reservation comes through here, and
// one that changes the entry it serves (its port of the Mac, its scheme,
// its audience, its Funnel port) unshares the old entry through
// unshareLocked before it changes, its service left unserved for the
// caller to share again. The caller holds d.mu.
func (d *Engine) rewriteLocked(ctx context.Context, i int, next ports.Reservation) {
	old := d.reservations[i]
	mode := d.servedModeLocked()
	if ports.ShareOf(old, mode) != ports.ShareOf(next, mode) {
		s := d.svcs[key(old.VM, old.GuestPort)]
		d.unshareLocked(ctx, old, mode, s != nil && s.served)
		if s != nil {
			s.served = false
		}
	}
	d.reservations[i] = next
	d.saveReservations()
}

// invariant: the one way a reservation's entry leaves the tailnet: at its
// end, and before any change to what it serves, a switch of the HTTPS mode
// included. While Serve is on, the entry it makes in mode goes, whether
// this engine served it or found it at start, never an entry at its port
// and path carried from another port of the Mac: one the owner put there
// by hand stays. One this engine did not serve goes only when the tailnet is
// read and carries it from its port of the Mac. The caller holds d.mu.
func (d *Engine) unshareLocked(ctx context.Context, l ports.Reservation, mode string, served bool) {
	if !d.self.On {
		return
	}
	want := ports.ShareOf(l, mode)
	shared, err := d.tn.Shared(ctx)
	if err != nil && !served {
		return
	}
	carried := slices.ContainsFunc(shared, func(e ports.Share) bool { return at(e, want) && ports.CarriedFrom(e) == want.Reserved })
	replaced := slices.ContainsFunc(shared, func(e ports.Share) bool { return at(e, want) && ports.CarriedFrom(e) != want.Reserved })
	if !served && !carried || served && replaced && !carried {
		return
	}
	_ = d.tn.Unshare(ctx, want)
}

// value: whether a tailnet entry answers at want's port and path.
func at(e, want ports.Share) bool {
	return e.Port == want.Port && e.Mounted() == want.Mounted() && (!want.Mounted() || e.Path == want.Path)
}

// value: the HTTPS mode this Mac's shares are served in: the one last
// served in, the preference's before any; the caller holds d.mu.
func (d *Engine) servedModeLocked() string {
	return cmp.Or(d.servedMode, d.policy().HTTPS)
}

// invariant: publication binds 127.0.0.1 only, never the tailnet ip or a
// wildcard: the tailnet carries its own bytes, so no firewall stands in the
// way; a reservation moves off a busy port and says so.
func (d *Engine) share(ctx context.Context, s *svc) {
	p := d.policy()
	s.detail, s.serveNote, s.refused = "", "", false
	i := d.reservationIndex(s.vm, s.port)
	if i < 0 {
		l, err := d.reserveLocked(p, s.vm, s.port, d.rights[s.vm], ports.ByRights)
		if err != nil {
			s.st.Phase, s.refused, s.detail = ports.PhaseRefused, true, err.Error()
			return
		}
		l.LastSeenAt = l.ReservedAt
		d.reservations = append(d.reservations, l)
		i = len(d.reservations) - 1
		d.saveReservations()
	}
	reservation := d.reservations[i]
	held, err := d.h.ListenLoopback(reservation.MacPort)
	if err != nil {
		was := reservation.MacPort
		others := append(append([]ports.Reservation(nil), d.reservations[:i]...), d.reservations[i+1:]...)
		moved, allocErr := ports.Reserve(p, others, s.vm, s.port, func(hp int) bool { return hp == was || d.busy(hp) })
		if allocErr != nil {
			s.st.Phase, s.refused, s.detail = ports.PhaseRefused, true, allocErr.Error()
			return
		}
		reservation.MacPort = moved.MacPort
		d.rewriteLocked(ctx, i, reservation)
		if held, err = d.h.ListenLoopback(reservation.MacPort); err != nil {
			s.detail = err.Error()
			return
		}
		s.detail = "reserved port " + strconv.Itoa(was) + " was busy: now " + strconv.Itoa(reservation.MacPort)
	}
	s.listeners = []Listener{held}
	go d.carry(ctx, s, held)
	if reservation.HTTP == nil {
		http := d.detectHTTP(ctx, s)
		reservation.HTTP = &http
		d.rewriteLocked(ctx, i, reservation)
	}
	d.expose(ctx, s, reservation)
}

// invariant: with Serve off no serve runs: the app would block waiting for the
// admin; a serve refused while Serve is on leaves the service local, its
// public URL unchanged, and says the tailnet's own line. A share is served
// in the mode last served in, never the preference's ahead of the tick
// that judges a switch, so it is unshared in the mode it was served in.
func (d *Engine) expose(ctx context.Context, s *svc, reservation ports.Reservation) {
	s.serveNote = ""
	if !d.self.On || reservation.Audience() == catalog.AudienceLocal {
		return
	}
	want := ports.ShareOf(reservation, d.servedModeLocked())
	if err := d.heldOutside(ctx, want); err != nil {
		s.serveNote = "serve: " + err.Error()
		return
	}
	if _, err := d.tn.Share(ctx, want); err != nil {
		s.serveNote = "serve: " + err.Error()
		if refused := (*ports.Refused)(nil); errors.As(err, &refused) {
			s.serveNote = "serve: " + refused.Detail
		}
		return
	}
	s.served = true
}

func (d *Engine) serveAll(ctx context.Context) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, s := range d.svcs {
		if i := d.reservationIndex(s.vm, s.port); i >= 0 && s.st.Phase == ports.PhaseShared && len(s.listeners) > 0 {
			d.expose(ctx, s, d.reservations[i])
		}
	}
}

func (d *Engine) unserveAll() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, s := range d.svcs {
		s.served, s.serveNote = false, ""
	}
}

func (d *Engine) unpublish(ctx context.Context, s *svc) {
	d.closeLocked(s)
	if i := d.reservationIndex(s.vm, s.port); i >= 0 && s.served {
		d.unshareLocked(ctx, d.reservations[i], d.servedModeLocked(), true)
	}
	s.served, s.serveNote = false, ""
}

func (d *Engine) closeLocked(s *svc) {
	for _, l := range s.listeners {
		_ = l.Close()
	}
	s.listeners = nil
}

func (d *Engine) clientOf(vm string) Guest {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.clientLocked(vm)
}

func (d *Engine) clientLocked(vm string) Guest {
	if v, ok := d.vms[vm]; ok {
		return v.client
	}
	return nil
}

func (d *Engine) carry(ctx context.Context, s *svc, l Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		go func() {
			client := d.clientOf(s.vm)
			if client == nil {
				_ = conn.Close()
				return
			}
			remote, err := client.Forward(ctx, ports.Dialed(s.ip), s.port)
			if err != nil {
				_ = conn.Close()
				return
			}
			d.h.Join(conn, remote)
		}()
	}
}

type readResult struct {
	n   int
	err error
}

// invariant: HTTP is detected once per reservation: a server that speaks first
// (ssh, mysql, smtp) is not HTTP; else a HEAD answered with "HTTP/" is.
func (d *Engine) detectHTTP(ctx context.Context, s *svc) bool {
	client := d.clientLocked(s.vm)
	if client == nil {
		return false
	}
	conn, err := client.Forward(ctx, ports.Dialed(s.ip), s.port)
	if err != nil {
		return false
	}
	defer func() { _ = conn.Close() }()
	buf := make([]byte, 64)
	got := make(chan readResult, 1)
	go func() {
		n, err := conn.Read(buf)
		got <- readResult{n, err}
	}()
	wait, cancel := context.WithTimeout(ctx, bannerWait)
	select {
	case r := <-got:
		cancel()
		return r.n >= 5 && string(buf[:5]) == "HTTP/"
	case <-wait.Done():
		cancel()
	}
	if _, err := conn.Write([]byte("HEAD / HTTP/1.0\r\n\r\n")); err != nil {
		return false
	}
	wait, cancel = context.WithTimeout(ctx, headWait)
	defer cancel()
	select {
	case r := <-got:
		return r.n >= 5 && string(buf[:5]) == "HTTP/"
	case <-wait.Done():
		return false
	}
}

// value: how this Mac shares, and why it shares nothing when it is off: a
// Mac signed out of the tailnet says so, not that Serve is off; the caller
// holds d.mu.
func (d *Engine) exposureLocked() ports.Exposure {
	e := ports.ExposureOf(d.self.On, d.self.DNSName)
	if !e.On && d.self.Backend != ports.Running {
		e.Off = catalog.SignedOutDetail
	}
	return e
}

func (d *Engine) view(l ports.Reservation, s *svc, httpsMode string) ports.SharedPort {
	v := ports.Hinted(ports.ViewOf(l, d.exposureLocked(), httpsMode, ports.PhaseWaiting), d.o.Ports)
	if s != nil {
		switch s.st.Phase {
		case ports.PhaseShared, ports.PhaseClosing:
			v.State = ports.PhaseShared
		case ports.PhaseRefused:
			v.State = ports.PhaseRefused
		case ports.PhaseNone, ports.PhaseWaiting, ports.PhaseGone, ports.PhaseUnknown, ports.PhaseNotAllowed:
		}
		v.Process, v.Bind = s.process, s.ip
		v.Detail = ports.JoinDetail(ports.JoinDetail(s.detail, s.serveNote), v.Detail)
		if d.issuing && s.served && l.HTTP != nil && *l.HTTP && ports.HTTPSLinks(httpsMode) {
			v.Detail = ports.JoinDetail(v.Detail, catalog.IssuingDetail)
		}
		if s.refused {
			v.State = ports.PhaseRefused
		}
	}
	if vs := d.vms[l.VM]; v.State == ports.PhaseWaiting && vs != nil && vs.state != engine.StateRunning {
		v.Detail = ports.JoinDetail(catalog.StoppedDetail(l.VM), v.Detail)
	}
	return v
}

func (d *Engine) macPorts(vm string) ports.MacPorts {
	p := d.policy()
	d.mu.Lock()
	defer d.mu.Unlock()
	e := d.exposureLocked()
	out := ports.MacPorts{
		Name: d.m.ThisMacName(), ThisMac: true, Reachable: true, Engine: ports.EngineInfo{Running: true, Version: d.o.Version},
		PublicName: e.PublicName, Serve: e.Word(), HTTPS: p.HTTPS, Shared: []ports.SharedPort{},
	}
	if vm == "" {
		out.Listening = d.listeningLocked()
	}
	off := ""
	if !e.On {
		off = e.Off
	}
	for _, l := range d.reservations {
		if vm == "" || l.VM == vm {
			out.Shared = append(out.Shared, d.view(l, d.svcs[key(l.VM, l.GuestPort)], p.HTTPS))
		}
	}
	for _, s := range d.svcs {
		if s.st.Phase == ports.PhaseRefused && d.reservationIndex(s.vm, s.port) < 0 && (vm == "" || s.vm == vm) {
			out.Shared = append(out.Shared, ports.SharedPort{VM: s.vm, GuestPort: s.port, State: ports.PhaseRefused, Process: s.process, Detail: ports.JoinDetail(s.detail, off)})
		}
		if s.st.Phase == ports.PhaseNotAllowed && (vm == "" || s.vm == vm) {
			out.Shared = append(out.Shared, ports.Hinted(ports.SharedPort{VM: s.vm, GuestPort: s.port, State: ports.PhaseNotAllowed, Process: s.process, Bind: s.ip, Detail: s.detail}, d.o.Ports))
		}
	}
	sort.Slice(out.Shared, func(i, j int) bool {
		a, b := out.Shared[i], out.Shared[j]
		if a.VM != b.VM {
			return a.VM < b.VM
		}
		return a.GuestPort < b.GuestPort
	})
	return out
}

// invariant: a port is reserved with the audience its VM's rights give it,
// and one the owner shares outside them is private; a public one takes its
// Funnel port, refused while another VM's share holds it, or when the HTTPS
// mode leaves none for it. by is who makes it, ports.ByOwner or
// ports.ByRights. The caller holds d.mu.
func (d *Engine) reserveLocked(p ports.Policy, vm string, port int, r ports.Rights, by string) (ports.Reservation, error) {
	l, err := ports.Reserve(p, d.reservations, vm, port, d.busy)
	if err != nil || d.reservationIndex(vm, port) >= 0 {
		return l, err
	}
	l.ReservedAt = d.h.Now().UTC().Truncate(time.Second)
	l.By = by
	judged, keep := r.Judge(l, p.HTTPS)
	switch {
	case !keep && r.Allows(port):
		return ports.Reservation{}, &ports.NoFunnelPort{Port: port}
	case !keep:
		return ports.Reservation{}, errors.New(ports.NotAllowed(r))
	}
	if err := d.funnelTakenLocked(judged); err != nil {
		return ports.Reservation{}, err
	}
	if h, ok := ports.HintOf(d.o.Ports, port); ok && h.HTTP {
		judged.HTTP = &h.HTTP
	}
	return judged, nil
}

func (d *Engine) funnelTakenLocked(l ports.Reservation) error {
	if l.Audience() != catalog.AudiencePublic {
		return nil
	}
	for _, o := range d.reservations {
		if o.Key() != l.Key() && o.Audience() == catalog.AudiencePublic && o.PublicPort == l.PublicPort {
			return &ports.FunnelTaken{Port: l.PublicPort, By: o.Key()}
		}
	}
	return nil
}

// invariant: station never replaces a tailnet entry it did not make: a
// public share is refused while its Funnel port serves another target,
// or is funneled with none of station's; a private share, a path or a
// port, is refused while its port is funneled, for Funnel opens every
// path of a port. The tailnet unread refuses both. The caller holds d.mu.
func (d *Engine) heldOutside(ctx context.Context, want ports.Share) error {
	funneled, err := d.tn.Funneled(ctx)
	if err != nil {
		return err
	}
	if !want.Public() {
		if slices.Contains(funneled, want.Port) {
			return &ports.FunnelHeld{Port: want.Port, Private: true}
		}
		return nil
	}
	shared, err := d.tn.Shared(ctx)
	if err != nil {
		return err
	}
	ours := false
	for _, e := range shared {
		if e.Port != want.Port {
			continue
		}
		if from := ports.CarriedFrom(e); from != want.Reserved {
			held := &ports.FunnelHeld{Port: want.Port}
			if d.policy().InRange(from) {
				held.From = from
			}
			return held
		}
		ours = true
	}
	if !ours && slices.Contains(funneled, want.Port) {
		return &ports.FunnelHeld{Port: want.Port}
	}
	return nil
}

// invariant: a VM's reservations are judged again by its rights each time
// they are read, at each of its runs and at its adoption: one the rights
// made that they no longer allow is unshared and released, its port
// flagged while it listens; one whose audience or Funnel port changed is
// unshared, then shared again with the new one. The caller holds d.mu.
func (d *Engine) judgeLocked(ctx context.Context, vm string, r ports.Rights, httpsMode string) {
	for _, l := range slices.Clone(d.reservations) {
		if l.VM != vm {
			continue
		}
		want, keep := r.Judge(l, httpsMode)
		if keep && d.funnelTakenLocked(want) != nil {
			want.To, want.PublicPort, keep = "", 0, want.Owned()
		}
		if keep && want.Audience() == l.Audience() && want.PublicPort == l.PublicPort {
			continue
		}
		s := d.svcs[key(vm, l.GuestPort)]
		live := s != nil && len(s.listeners) > 0
		if keep {
			if s != nil {
				d.closeLocked(s)
				s.serveNote = ""
			}
			d.rewriteLocked(ctx, d.reservationIndex(vm, l.GuestPort), want)
			if live {
				d.share(ctx, s)
			}
			continue
		}
		d.dropLocked(ctx, vm, l.GuestPort)
		switch {
		case s == nil:
		case !live && s.st.Phase != ports.PhaseShared && s.st.Phase != ports.PhaseClosing:
			delete(d.svcs, key(vm, l.GuestPort))
		case r.Allows(l.GuestPort):
			s.st.Phase, s.refused, s.detail = ports.PhaseRefused, true, (&ports.NoFunnelPort{Port: l.GuestPort}).Error()
		default:
			s.st.Phase, s.detail = ports.PhaseNotAllowed, ports.NotAllowed(r)
		}
	}
}

// invariant: a Mac signed out of the tailnet refuses a share the owner asks
// for.
func (d *Engine) reserve(ctx context.Context, vm string, port int, by string) (ports.SharedPort, error) {
	if _, err := d.find(vm); err != nil {
		return ports.SharedPort{}, err
	}
	r := d.rightsOf(ctx, vm)
	p := d.policy()
	d.mu.Lock()
	defer d.mu.Unlock()
	l, err := d.reserveLocked(p, vm, port, r, by)
	if err != nil {
		return ports.SharedPort{}, asOp(err)
	}
	if i := d.reservationIndex(vm, port); i < 0 {
		d.reservations = append(d.reservations, l)
		d.saveReservations()
	} else if by == ports.ByOwner && !l.Owned() {
		l.By = ports.ByOwner
		d.rewriteLocked(ctx, i, l)
	}
	s := d.svcs[key(vm, port)]
	if s != nil && (s.st.Phase == ports.PhaseRefused || s.st.Phase == ports.PhaseNotAllowed || s.refused) {
		s.st.Phase = ports.PhaseNone
		var acts []ports.Action
		s.st, acts = ports.Next(s.st, ports.EvListen, d.h.Now())
		d.apply(ctx, s, acts)
	}
	if d.self.Backend != ports.Running {
		return ports.SharedPort{}, ports.SignedOut(d.m.ThisMacName())
	}
	return d.view(l, s, p.HTTPS), nil
}

func (d *Engine) release(ctx context.Context, vm string, port int) (ports.Reservation, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	i := d.reservationIndex(vm, port)
	if i < 0 {
		return ports.Reservation{}, ports.NotFoundAt(ports.Target{VM: vm, Port: port})
	}
	l := d.reservations[i]
	d.dropLocked(ctx, vm, port)
	delete(d.svcs, key(vm, port))
	return l, nil
}

// invariant: station owns only the serve entries of its range (port mode and
// raw TCP), and the path entries /<vm>/<port> of 443 (path mode) and the
// Funnel entries carried from its range, its VM known or not, and removes
// those no reservation justifies in the HTTPS mode of this Mac, every kind;
// a reservation its VM's rights no longer allow justifies none, for they are
// judged first. It never touches another entry, and waits for Serve to be
// on.
func (d *Engine) reconcileServe(ctx context.Context) {
	d.mu.Lock()
	on := d.self.On
	judged := map[string]bool{}
	for _, l := range d.reservations {
		judged[l.VM] = true
	}
	d.mu.Unlock()
	if !on {
		return
	}
	for vm := range judged {
		d.rightsOf(ctx, vm)
	}
	shared, err := d.tn.Shared(ctx)
	if err != nil {
		return
	}
	p := d.policy()
	d.mu.Lock()
	reservations := append([]ports.Reservation(nil), d.reservations...)
	d.mu.Unlock()
	justified := func(e ports.Share) bool {
		return slices.ContainsFunc(reservations, func(l ports.Reservation) bool { return justifies(l, e, p.HTTPS) })
	}
	for _, e := range shared {
		switch {
		case e.Mounted():
			parts := strings.Split(strings.Trim(e.Path, "/"), "/")
			if e.Port != 443 || len(parts) < 2 || !p.InRange(ports.CarriedFrom(e)) {
				continue
			}
			if !justified(e) {
				_ = d.tn.Unshare(ctx, e)
				continue
			}
			d.servedBefore(ports.CarriedFrom(e))
		case e.Public():
			if !p.InRange(ports.CarriedFrom(e)) {
				continue
			}
			if !justified(e) {
				_ = d.tn.Unshare(ctx, e)
				continue
			}
			d.servedBefore(ports.CarriedFrom(e))
		case p.InRange(e.Port):
			if !justified(e) {
				e.Path = ""
				_ = d.tn.Unshare(ctx, e)
				continue
			}
			d.servedBefore(e.Port)
		}
	}
}

// value: whether a tailnet entry is the share a reservation makes in this
// HTTPS mode: the same port, scheme, audience and path, carried from its
// port of the Mac; a local one makes none.
func justifies(l ports.Reservation, e ports.Share, httpsMode string) bool {
	if l.Audience() == catalog.AudienceLocal {
		return false
	}
	s := ports.ShareOf(l, httpsMode)
	if s.Port != e.Port || s.Scheme != e.Scheme || s.Mounted() != e.Mounted() || s.Mounted() && s.Path != e.Path {
		return false
	}
	if !e.Mounted() && s.Public() != e.Public() {
		return false
	}
	from := ports.CarriedFrom(e)
	return from == 0 || from == l.MacPort
}

// invariant: a reserved port the tailnet already serves when the engine
// starts, by any entry it justifies (a private port, a path, a Funnel
// port), is served until its VM says it no longer listens there, so a
// release, a stop or a judge removes that entry.
func (d *Engine) servedBefore(macPort int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, l := range d.reservations {
		if l.MacPort == macPort {
			if s := d.svcOf(l.VM, l.GuestPort); s.st.Phase != ports.PhaseShared {
				s.served = true
			}
		}
	}
}
