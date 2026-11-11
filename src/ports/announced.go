package ports

import (
	"sort"
	"strings"

	"github.com/jterrazz/studio-station/src/catalog"
)

// value: who a listener a guest announces was read from: a socket of the
// VM, or a container's published port.
const (
	BySocket    = "socket"
	ByContainer = catalog.ContainerSource
)

type Announced struct {
	// value: one port that listens inside a VM, as its guest announces it
	// to the host.
	Port    int    `json:"port"`
	IP      string `json:"ip"`
	Process string `json:"process"`
	PID     int    `json:"pid"`
	Source  string `json:"source"`
	// value: the listener sits in the guest's ephemeral range: the host does
	// not list it, nor judge it, until an owner shares it.
	Quiet bool `json:"quiet,omitempty"`
}

// value: the range a guest's system hands its outgoing connections' ports
// from, as the kind reads it when the guest cannot.
const (
	LinuxEphemeralFrom, LinuxEphemeralTo = 32768, 60999
	MacOSEphemeralFrom, MacOSEphemeralTo = 49152, 65535
)

func MarkQuiet(ls []Announced, from, to int) []Announced {
	// invariant: a socket on a port of the guest's ephemeral range is the
	// guest's own business, never a service: it is marked, not dropped, so an
	// owner's `ports share` still reaches it.
	out := make([]Announced, len(ls))
	for i, l := range ls {
		l.Quiet = l.Source == BySocket && l.Port >= from && l.Port <= to
		out[i] = l
	}
	return out
}

type Announcement struct {
	Listen   *Announced
	Unlisten int
}

type VMLivesOn struct {
	// value: alias of guest.json v1, a VM's placement as its Mac told its
	// guest: the Mac it runs on, its own name, the public name of its ports.
	Mac        string `json:"host"`
	VM         string `json:"vm"`
	PublicName string `json:"public_host"`
}

func Announce(ls []Listener) []Announced {
	// invariant: one listener per port: an IPv4 address wins its port over an
	// IPv6 one, and a later address of the same family over an earlier one.
	byPort := map[int]Announced{}
	for _, l := range ls {
		if held, seen := byPort[l.Port]; seen && !strings.Contains(held.IP, ":") && l.IPv6() {
			continue
		}
		byPort[l.Port] = Announced{Port: l.Port, IP: l.Bind, Process: l.Process, PID: l.PID, Source: BySocket}
	}
	return SortedAnnounced(byPort)
}

func SortedAnnounced(byPort map[int]Announced) []Announced {
	out := make([]Announced, 0, len(byPort))
	for _, l := range byPort {
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Port < out[j].Port })
	return out
}

func MergeAnnounced(sockets, containers []Announced) []Announced {
	// invariant: a container's published port wins its port over the socket
	// that carries it.
	byPort := map[int]Announced{}
	for _, l := range sockets {
		byPort[l.Port] = l
	}
	for _, l := range containers {
		byPort[l.Port] = l
	}
	return SortedAnnounced(byPort)
}

func FilterAnnounced(ls []Announced, minPort int, ignored []int) []Announced {
	out := ls[:0:0]
	for _, l := range ls {
		skip := l.Port < minPort
		for _, p := range ignored {
			skip = skip || p == l.Port
		}
		if !skip {
			out = append(out, l)
		}
	}
	return out
}

func DiffAnnounced(before, after []Announced) []Announcement {
	was := map[int]Announced{}
	for _, l := range before {
		was[l.Port] = l
	}
	var out []Announcement
	now := map[int]bool{}
	for _, l := range after {
		now[l.Port] = true
		if old, ok := was[l.Port]; !ok || old.IP != l.IP || old.Process != l.Process || old.Source != l.Source || old.Quiet != l.Quiet {
			listen := l
			out = append(out, Announcement{Listen: &listen})
		}
	}
	for _, l := range before {
		if !now[l.Port] {
			out = append(out, Announcement{Unlisten: l.Port})
		}
	}
	return out
}
