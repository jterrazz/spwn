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
			if i := d.reservationIndex(s.vm, s.port); i >= 0 {
				d.reservations = append(d.reservations[:i:i], d.reservations[i+1:]...)
				d.saveReservations()
			}
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
// mode it was served in; the caller holds d.mu.
func (d *Engine) serveInModeLocked(ctx context.Context, mode string) {
	was := d.servedMode
	d.servedMode = mode
	if was == "" || was == mode || !d.self.On {
		return
	}
	for _, s := range d.svcs {
		i := d.reservationIndex(s.vm, s.port)
		if i < 0 || !s.served || d.reservations[i].Audience() != catalog.AudiencePrivate {
			continue
		}
		_ = d.tn.Unshare(ctx, ports.ShareOf(d.reservations[i], was))
		s.served = false
		d.expose(ctx, s, d.reservations[i], mode)
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
	for _, s := range d.svcs {
		if s.vm == vm {
			d.unpublish(ctx, s)
			delete(d.svcs, key(s.vm, s.port))
		}
	}
	httpsMode := d.policy().HTTPS
	kept, gone := ports.Without(d.reservations, func(l ports.Reservation) bool { return l.VM != vm })
	if d.self.On {
		for _, l := range gone {
			_ = d.tn.Unshare(ctx, ports.ShareOf(l, httpsMode))
		}
	}
	if len(gone) > 0 {
		d.reservations = kept
		d.saveReservations()
	}
}

// invariant: publication binds 127.0.0.1 only, never the tailnet ip or a
// wildcard: the tailnet carries its own bytes, so no firewall stands in the
// way; a reservation moves off a busy port and says so.
func (d *Engine) share(ctx context.Context, s *svc) {
	p := d.policy()
	s.detail, s.serveNote, s.refused = "", "", false
	i := d.reservationIndex(s.vm, s.port)
	if i < 0 {
		l, err := d.reserveLocked(p, s.vm, s.port, d.rights[s.vm])
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
		d.reservations[i] = reservation
		d.saveReservations()
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
		d.reservations[i] = reservation
		d.saveReservations()
	}
	d.expose(ctx, s, reservation, p.HTTPS)
}

// invariant: with Serve off no serve runs: the app would block waiting for the
// admin; a serve refused while Serve is on leaves the service local, its
// public URL unchanged, and says the tailnet's own line.
func (d *Engine) expose(ctx context.Context, s *svc, reservation ports.Reservation, httpsMode string) {
	s.serveNote = ""
	if !d.self.On || reservation.Audience() == catalog.AudienceLocal {
		return
	}
	if _, err := d.tn.Share(ctx, ports.ShareOf(reservation, httpsMode)); err != nil {
		s.serveNote = "serve: " + err.Error()
		if refused := (*ports.Refused)(nil); errors.As(err, &refused) {
			s.serveNote = "serve: " + refused.Detail
		}
		return
	}
	s.served = true
}

func (d *Engine) serveAll(ctx context.Context) {
	p := d.policy()
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, s := range d.svcs {
		if i := d.reservationIndex(s.vm, s.port); i >= 0 && s.st.Phase == ports.PhaseShared && len(s.listeners) > 0 {
			d.expose(ctx, s, d.reservations[i], p.HTTPS)
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
	for _, l := range s.listeners {
		_ = l.Close()
	}
	s.listeners = nil
	if s.served && d.self.On {
		if i := d.reservationIndex(s.vm, s.port); i >= 0 {
			_ = d.tn.Unshare(ctx, ports.ShareOf(d.reservations[i], d.policy().HTTPS))
		}
	}
	s.served, s.serveNote = false, ""
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
// Funnel port, refused while another VM's share holds it. The caller holds
// d.mu.
func (d *Engine) reserveLocked(p ports.Policy, vm string, port int, r ports.Rights) (ports.Reservation, error) {
	l, err := ports.Reserve(p, d.reservations, vm, port, d.busy)
	if err != nil || d.reservationIndex(vm, port) >= 0 {
		return l, err
	}
	l.ReservedAt = d.h.Now().UTC().Truncate(time.Second)
	if r.Allows(port) && r.Audience() != catalog.AudiencePrivate {
		l.To, l.PublicPort = r.Audience(), r.PublicPort(port)
	}
	if l.To == catalog.AudiencePublic {
		for _, o := range d.reservations {
			if o.Audience() == catalog.AudiencePublic && o.PublicPort == l.PublicPort {
				return ports.Reservation{}, &ports.FunnelTaken{Port: l.PublicPort, By: o.Key()}
			}
		}
	}
	if h, ok := ports.HintOf(d.o.Ports, port); ok && h.HTTP {
		l.HTTP = &h.HTTP
	}
	return l, nil
}

// invariant: a Mac signed out of the tailnet refuses a share the owner asks
// for.
func (d *Engine) reserve(ctx context.Context, vm string, port int) (ports.SharedPort, error) {
	if _, err := d.find(vm); err != nil {
		return ports.SharedPort{}, err
	}
	r := d.rightsOf(ctx, vm)
	p := d.policy()
	d.mu.Lock()
	defer d.mu.Unlock()
	l, err := d.reserveLocked(p, vm, port, r)
	if err != nil {
		return ports.SharedPort{}, asOp(err)
	}
	if i := d.reservationIndex(vm, port); i < 0 {
		d.reservations = append(d.reservations, l)
		d.saveReservations()
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
	if s, ok := d.svcs[key(vm, port)]; ok {
		d.unpublish(ctx, s)
		delete(d.svcs, key(vm, port))
	}
	d.reservations = append(d.reservations[:i:i], d.reservations[i+1:]...)
	d.saveReservations()
	return l, nil
}

// invariant: station owns only the serve entries of its range (port mode and
// raw TCP), of a path /<vm>/ of this host (path mode), and the Funnel
// entries carried from its range, and removes those no reservation
// justifies in the HTTPS mode of this Mac, every kind; it never touches
// another entry, and waits for Serve to be on.
func (d *Engine) reconcileServe(ctx context.Context) {
	d.mu.Lock()
	on := d.self.On
	d.mu.Unlock()
	if !on {
		return
	}
	shared, err := d.tn.Shared(ctx)
	if err != nil {
		return
	}
	p := d.policy()
	d.mu.Lock()
	reservations := append([]ports.Reservation(nil), d.reservations...)
	vms := map[string]bool{}
	for name := range d.vms {
		vms[name] = true
	}
	d.mu.Unlock()
	for _, e := range shared {
		switch {
		case e.Public():
			justified := false
			for _, l := range reservations {
				justified = justified || l.Audience() == catalog.AudiencePublic && l.PublicPort == e.Port
			}
			if !justified && p.InRange(ports.CarriedFrom(e)) {
				_ = d.tn.Unshare(ctx, e)
			}
		case p.InRange(e.Port):
			justified := false
			for _, l := range reservations {
				justified = justified || l.MacPort == e.Port && ports.ShareOf(l, p.HTTPS).Scheme == e.Scheme
			}
			if !justified {
				e.Path = ""
				_ = d.tn.Unshare(ctx, e)
				continue
			}
			d.servedBefore(e.Port)
		case e.Port == 443 && strings.Count(e.Path, "/") >= 2:
			parts := strings.Split(strings.Trim(e.Path, "/"), "/")
			justified := false
			for _, l := range reservations {
				justified = justified || (p.HTTPS == ports.HTTPSPath && l.VM == parts[0] && strconv.Itoa(l.GuestPort) == parts[len(parts)-1])
			}
			if vms[parts[0]] && !justified {
				_ = d.tn.Unshare(ctx, e)
			}
		}
	}
}

// invariant: a reserved port the tailnet already serves when the engine
// starts is served until its VM says it no longer listens there.
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
