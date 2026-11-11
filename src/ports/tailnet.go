package ports

import (
	"errors"

	"github.com/jterrazz/studio-station/src/catalog"
)

const (
	SchemeHTTPS = "https"
	SchemeTCP   = "tcp"
)

const (
	VariantStandalone = "standalone"
	VariantAppStore   = "app-store"
)

const Running = "Running"

type Node struct {
	ID, Name, DNSName, IP string
	Online                bool
	Tags                  []string
}

type Client struct{ Path, Variant string }

type SharingState struct {
	Backend string
	// invariant: sharing is on when the tailnet issues certificates to a node
	// that runs: a share needs HTTPS certificates, and a node logged out or
	// absent shares nothing.
	On      bool
	DNSName string
}

type Self struct {
	Node
	Sharing        SharingState
	Login, Network string
	Client         Client
	// value: the other nodes this one sees, read in the same answer as itself.
	Peers []Node
}

type Share struct {
	// value: where the tailnet answers it: Reserved, or 443 under Path.
	Port int
	Path string
	// value: the port on this Mac's loopback that carries it.
	Reserved int
	Scheme   string
	// value: what the tailnet forwards to, as it reports it.
	Target string
	// value: who it is opened to: private on the tailnet, public on the
	// internet (Port is then a Funnel port); empty is private.
	To catalog.Audience `json:",omitempty"`
}

func (s Share) Public() bool { return s.To == catalog.AudiencePublic }

func (s Share) Mounted() bool { return s.Path != "" && s.Path != "/" }

type Absent struct{ App string }

func (e *Absent) Error() string { return e.App + " is not installed" }

type Refused struct{ By, Detail string }

func (e *Refused) Error() string { return e.By + " refused: " + e.Detail }

type Served struct {
	Share
	Station bool
}

type Status struct {
	State        catalog.State
	Connected    bool
	BackendState string
	Name, IP     string
	DNSName      string
	Variant, CLI string
	ServeOn      bool
	Peers        []Node
	Serve        []Served
}

func StatusOf(self Self, err error, shared []Share, owns func(Share) bool) Status {
	// invariant: an app that is down or absent is a state, never an error:
	// absent when no client answers, degraded when one answers nothing readable
	// or its node does not run.
	out := Status{State: catalog.StateAbsent, Variant: self.Client.Variant, CLI: self.Client.Path}
	if err != nil {
		var absent *Absent
		if !errors.As(err, &absent) {
			out.State = catalog.StateDegraded
		}
		return out
	}
	out.BackendState = self.Sharing.Backend
	out.Connected = self.Sharing.Backend == Running
	out.ServeOn = self.Sharing.On
	out.State = catalog.StateDegraded
	if out.Connected {
		out.State = catalog.StateOK
	}
	out.Name, out.IP, out.DNSName = self.Name, self.IP, self.DNSName
	out.Peers = self.Peers
	for _, s := range shared {
		out.Serve = append(out.Serve, Served{Share: s, Station: owns != nil && owns(s)})
	}
	return out
}
