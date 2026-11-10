package wire

import (
	"context"

	awakecontract "github.com/jterrazz/studio-station/src/awake/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	coreadapterlinux "github.com/jterrazz/studio-station/src/core/adapter/linux"
	coreadaptermacos "github.com/jterrazz/studio-station/src/core/adapter/macos"
	"github.com/jterrazz/studio-station/src/core/contract"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/platform/local"
)

// invariant: the system STATION_MACHINE_KIND pins reads itself with its own
// adapter, so a pinned Mac is read as a Mac on every host the binary runs on.
func selfOf(h local.System, pin string) contract.Self {
	system := h.Env.GOOS()
	if _, os, ok := pinned(pin); ok {
		system = string(os)
	}
	if system == "linux" {
		return coreadapterlinux.New(h.Env.Hostname())
	}
	return coreadaptermacos.New(h.Run, h.Env.Hostname())
}

// value: the kind STATION_MACHINE_KIND pins, in the words it is spelled
// with; a spec pins it so a document means the same on every system, and Self
// is then not asked.
func pinned(word string) (coremachine.Kind, coremachine.OS, bool) {
	switch catalog.MachineKind(word) {
	case catalog.Mac:
		return coremachine.Mac, coremachine.MacOS, true
	case catalog.MacOS:
		return coremachine.Workspace, coremachine.MacOS, true
	case catalog.Linux:
		return coremachine.Workspace, coremachine.Linux, true
	}
	return "", "", false
}

// invariant: the Profile every rule reads is built once, here: the kind and
// the system from Self (or the pin), the battery, lid and thermal from
// Power, the job manager, the hypervisor and the engine from which adapters
// this machine is wired with (launchd and tart on macOS, an engine on a Mac).
func profileOf(ctx context.Context, self contract.Self, pin, name string, power awakecontract.Power, hypervisor func() (string, bool)) coremachine.Profile {
	var p coremachine.Profile
	if kind, os, ok := pinned(pin); ok {
		p.Kind, p.System.OS = kind, os
	} else {
		_, p.Kind, p.System, p.Hardware, _ = self.Read(ctx)
	}
	p.Ref = coremachine.Ref{Name: name}
	mac := p.Kind == coremachine.Mac
	p.Caps = coremachine.Capabilities{Jobs: p.System.OS == coremachine.MacOS, Engine: mac}
	if mac {
		name, installed := hypervisor()
		p.Caps.Hypervisor, p.Caps.VMs = name, installed
	}
	if mac {
		p.Caps = power.Capabilities().Onto(p.Caps)
	}
	return p
}

// value: the kind the packages not yet on the Profile read: a Mac is a
// system, a VM is named by its system.
func legacyKind(p coremachine.Profile) catalog.MachineKind {
	switch {
	case p.Kind == coremachine.Mac:
		return catalog.Mac
	case p.System.OS == coremachine.Linux:
		return catalog.Linux
	}
	return catalog.MacOS
}
