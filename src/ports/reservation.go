package ports

import (
	"cmp"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
)

type Reservation struct {
	// value: alias of station.leases/v1's entry, host_port included, until
	// its store migrates to reservations.json (ARCHITECTURE §12).
	VM        string `json:"vm"`
	GuestPort int    `json:"guest_port"`
	MacPort   int    `json:"host_port"`
	// value: who it is shared with, as the rights said when it was
	// reserved; empty is private, every reservation made before rights.
	To catalog.Audience `json:"to,omitempty"`
	// value: the Funnel port a public one is shared on.
	PublicPort int       `json:"public_port,omitempty"`
	HTTP       *bool     `json:"http,omitempty"`
	ReservedAt time.Time `json:"reserved_at"`
	LastSeenAt time.Time `json:"last_seen_at,omitzero"`
}

func (r Reservation) Key() string { return r.VM + ":" + strconv.Itoa(r.GuestPort) }

func (r Reservation) Audience() catalog.Audience { return cmp.Or(r.To, catalog.AudiencePrivate) }

type Declaration struct {
	// value: a port of a machine declared public, and the route that reaches it.
	Machine string `json:"machine"`
	Port    int    `json:"port"`
	Via     string `json:"via"`
	// value: the link it is reached by from outside (`https://jterrazz.com`);
	// empty when only its address is known.
	URL string `json:"url,omitempty"`
	// value: the setting that declared it; empty when the owner did.
	By string `json:"by,omitempty"`
}

func (d Declaration) Key() string { return d.Machine + ":" + strconv.Itoa(d.Port) }

func (d Declaration) Domain() (domain string, web bool) {
	// value: where its link reaches, as an address pill says it
	// (`jterrazz.com`), and whether a browser opens it; empty when it has
	// no link.
	rest, web := strings.CutPrefix(d.URL, "https://")
	if !web {
		rest, web = strings.CutPrefix(d.URL, "http://")
	}
	if !web {
		_, rest, _ = strings.Cut(d.URL, "://")
	}
	domain, _, _ = strings.Cut(rest, "/")
	return domain, web
}

type PortsGroup struct {
	Range    string `json:"range,omitempty"`
	MaxPerVM int    `json:"max_per_vm,omitempty"`
	HTTPS    string `json:"https,omitempty"`
	// value: the ports the owner stopped sharing, as <vm>:<port>.
	NotShared []string `json:"not_shared,omitempty"`
}
