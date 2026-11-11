package ports

import (
	"errors"
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
)

var ErrEngineDown = errors.New("the engine is not running")

const (
	ServeOn  = "on"
	ServeOff = "off"
)

type Exposure struct {
	On         bool
	PublicName string
	// value: why nothing is shared when it is off.
	Off string
}

func ExposureOf(serveOn bool, dnsName string) Exposure {
	if !serveOn || dnsName == "" {
		return Exposure{PublicName: Loopback, Off: catalog.ServeOffDetail}
	}
	return Exposure{On: true, PublicName: dnsName}
}

func (e Exposure) Word() string {
	if e.On {
		return ServeOn
	}
	return ServeOff
}

func ViewOf(l Reservation, e Exposure, httpsMode string, phase Phase) SharedPort {
	v := SharedPort{
		VM: l.VM, GuestPort: l.GuestPort, Reserved: l.MacPort, State: phase, To: string(l.Audience()),
		HTTP: l.HTTP != nil && *l.HTTP, URLs: URLs(l, e.PublicName, httpsMode, e.On), ReservedAt: l.ReservedAt,
	}
	if !l.LastSeenAt.IsZero() {
		seen := l.LastSeenAt
		v.LastSeenAt = &seen
	}
	if !e.On && l.Audience() != catalog.AudienceLocal {
		v.Detail = e.Off
	}
	return v
}

func JoinDetail(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	}
	return a + ", " + b
}

func ParsePort(text string) (int, error) {
	n, err := strconv.Atoi(text)
	if err != nil || n < 1 || n > 65535 {
		return 0, &BadPortError{Value: text}
	}
	return n, nil
}

func NotFoundAt(t Target) error { return NotFound(t.VM, t.Port) }

const ReservationsSchema = "station.services/v1"

func Find(reservations []Reservation, vm string, port int) (found Reservation, index int) {
	for i, l := range reservations {
		if l.VM == vm && l.GuestPort == port {
			return l, i
		}
	}
	return Reservation{}, -1
}

func Without(reservations []Reservation, keep func(Reservation) bool) (kept, gone []Reservation) {
	for _, l := range reservations {
		if keep(l) {
			kept = append(kept, l)
		} else {
			gone = append(gone, l)
		}
	}
	return kept, gone
}

type Event int

const (
	EvListen Event = iota
	EvUnlisten
	EvTick
	EvVMStopped
	EvDeleted
	EvReleased
)

type Action int

const (
	ActShare Action = iota
	ActUnpublish
	ActRelease
)

type State struct {
	Phase      Phase
	Since      time.Time
	LastSeenAt time.Time
}

func Next(st State, ev Event, now time.Time) (State, []Action) {
	// invariant: the lifecycle of one service (N5): a stop of listening unpublishes
	// only after the grace, a re-listen inside it keeps everything; a lease outlives
	// its listener and idles out after 7 days; a port not allowed is never
	// shared by a listen.
	switch ev {
	case EvListen:
		st.LastSeenAt = now
		switch st.Phase {
		case PhaseClosing:
			st.Phase = PhaseShared
			return st, nil
		case PhaseShared:
			return st, nil
		case PhaseNotAllowed:
			return st, nil
		case PhaseNone, PhaseWaiting, PhaseRefused, PhaseGone, PhaseUnknown:
		}
		st.Phase, st.Since = PhaseShared, now
		return st, []Action{ActShare}
	case EvUnlisten:
		if st.Phase == PhaseShared {
			st.Phase, st.Since = PhaseClosing, now
		}
		return st, nil
	case EvTick:
		switch {
		case st.Phase == PhaseClosing && now.Sub(st.Since) >= catalog.PortPolicy.Grace:
			st.Phase, st.Since = PhaseWaiting, now
			return st, []Action{ActUnpublish}
		case st.Phase == PhaseWaiting && !st.LastSeenAt.IsZero() && now.Sub(st.LastSeenAt) > catalog.PortPolicy.Idle:
			st.Phase = PhaseGone
			return st, []Action{ActRelease}
		}
		return st, nil
	case EvVMStopped:
		if st.Phase == PhaseShared || st.Phase == PhaseClosing {
			st.Phase, st.Since = PhaseWaiting, now
			return st, []Action{ActUnpublish}
		}
		if st.Phase == PhaseNone {
			st.Phase = PhaseWaiting
		}
		return st, nil
	case EvDeleted, EvReleased:
		was := st.Phase
		st.Phase = PhaseGone
		if was == PhaseShared || was == PhaseClosing {
			return st, []Action{ActUnpublish, ActRelease}
		}
		return st, []Action{ActRelease}
	}
	return st, nil
}

const Loopback = "127.0.0.1"

func isHTTP(l Reservation) bool { return l.HTTP == nil || *l.HTTP }

func URLs(l Reservation, publicName, httpsMode string, serveOn bool) URLSet {
	// invariant: one shape everywhere. An HTTP service, or one not yet detected,
	// carries a scheme so a URL parser reads a host; with Serve off the local URL
	// is the only one there is; a raw TCP service is the tailnet's bytes, no scheme.
	port := strconv.Itoa(l.MacPort)
	u := URLSet{Local: Loopback + ":" + port}
	if isHTTP(l) {
		u.Local = "http://" + u.Local
	}
	u.Public = u.Local
	if !serveOn || publicName == "" || l.Audience() == catalog.AudienceLocal {
		return u
	}
	switch {
	case l.Audience() == catalog.AudiencePublic && !isHTTP(l):
		u.Public = publicName + ":" + strconv.Itoa(l.PublicPort)
	case l.Audience() == catalog.AudiencePublic:
		u.Public = "https://" + publicName + ":" + strconv.Itoa(l.PublicPort)
	case !isHTTP(l):
		u.Public = publicName + ":" + port
	case httpsMode == HTTPSOff:
		u.Public = "http://" + publicName + ":" + port
	case httpsMode == HTTPSPath:
		u.Public = "https://" + publicName + "/" + l.VM + "/" + strconv.Itoa(l.GuestPort) + "/"
	default:
		u.Public = "https://" + publicName + ":" + port
	}
	return u
}

func ShareOf(l Reservation, httpsMode string) Share {
	// invariant: a public share is opened on its Funnel port, https for an
	// HTTP server and raw for any other, whatever the HTTPS mode: Funnel
	// answers on its own ports only.
	s := Share{Port: l.MacPort, Reserved: l.MacPort, Scheme: SchemeHTTPS, To: l.Audience()}
	switch {
	case s.Public():
		s.Port = l.PublicPort
		if !isHTTP(l) {
			s.Scheme = SchemeTCP
		}
	case !isHTTP(l), httpsMode == HTTPSOff:
		s.Scheme = SchemeTCP
	case httpsMode == HTTPSPath:
		s.Port, s.Path = 443, "/"+l.VM+"/"+strconv.Itoa(l.GuestPort)
	}
	return s
}

func Dialed(ip string) string {
	// invariant: guest-reported wildcards map to the loopback of their family, so
	// a tool bound to 127.0.0.1 in the VM is reached without --host.
	switch ip {
	case "", "0.0.0.0":
		return "127.0.0.1"
	case "::":
		return "::1"
	}
	return ip
}

type BadPortError struct{ Value string }

func (e *BadPortError) Error() string { return "\"" + e.Value + "\" is not a port" }

type IgnoredPortError struct{ Port int }

func (e *IgnoredPortError) Error() string {
	return "port " + strconv.Itoa(e.Port) + " is never shared"
}

type MacAwayError struct{ Mac string }

func (e *MacAwayError) Error() string {
	if e.Mac == "" {
		return "no Mac has connected to this VM yet"
	}
	return e.Mac + " is not connected to this VM"
}

func (e *MacAwayError) Hint() string {
	if e.Mac == "" {
		return "on the VM's Mac: station engine start"
	}
	return "on " + e.Mac + ": station engine start"
}

type UnknownVMError struct{ Name string }

func (e *UnknownVMError) Error() string { return "unknown VM \"" + e.Name + "\"" }
