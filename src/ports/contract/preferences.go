package contract

import (
	"context"
	"encoding/json"

	core "github.com/jterrazz/studio-station/src/core/contract"
	"github.com/jterrazz/studio-station/src/core/store"
	"github.com/jterrazz/studio-station/src/ports"
)

const GroupPorts = "ports"

const (
	// value: alias of the key station v2 kept the ports policy under, read
	// under ports and written there at the next write.
	RetiredGroupPorts = "services"
)

type PortsPrefs struct{ p core.Preferences }

func Ports(p core.Preferences) PortsPrefs { return PortsPrefs{p: p} }

func (s PortsPrefs) Get() (*ports.PortsGroup, error) {
	// invariant: nil when the owner chose nothing, so every default holds.
	var g *ports.PortsGroup
	err := s.p.Read(GroupPorts, &g)
	return g, err
}

func (s PortsPrefs) Set(g *ports.PortsGroup) error { return s.p.Write(GroupPorts, g) }

func (s PortsPrefs) Migrate(context.Context) error {
	// invariant: the retired keys leave the ports group, every other key of
	// it kept as it was; a group without them is left byte for byte. Their
	// values are not carried: a VM's ports are its Machinefile's rights, and
	// one with no Machinefile shares as their defaults did.
	var raw map[string]json.RawMessage
	if err := s.p.Read(GroupPorts, &raw); err != nil {
		return store.OnlyNewer(err)
	}
	retired := false
	// value: the keys a station before rights kept, which said which of a
	// VM's ports were shared: a Machinefile's ports say it now.
	for _, k := range [...]string{"min_port", "ignored_ports"} {
		if _, ok := raw[k]; ok {
			delete(raw, k)
			retired = true
		}
	}
	if !retired {
		return nil
	}
	return s.p.Write(GroupPorts, raw)
}
