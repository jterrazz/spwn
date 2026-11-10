package wire

import (
	"slices"

	"github.com/jterrazz/studio-station/src/accounts/adapter/studio"
	"github.com/jterrazz/studio-station/src/catalog"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/ports/adapter/tailscale"
)

// invariant: what provides each need on this Mac is read once, when its
// Profile is built, by the presence its adapter answers (a path looked up, a
// file seen), never by a package manager's listing; a world's Mac lacks the
// providers it lists absent, and its hypervisor is its own fake's.
func (m machine) providers() map[coremachine.Need]coremachine.Provider {
	hv := m.hypervisor()
	tn := m.tailnetCLI()
	looks := map[coremachine.Need]func() bool{
		coremachine.Hypervisor:     hv.Installed,
		coremachine.PrivateNetwork: tn.Present,
		coremachine.Credentials:    func() bool { return studio.Present(m.sys.Run) },
	}
	names := map[coremachine.Need]string{
		coremachine.Hypervisor:     hv.Name(),
		coremachine.PrivateNetwork: tailscale.App,
		coremachine.Credentials:    studio.Command,
	}
	out := map[coremachine.Need]coremachine.Provider{}
	for need, look := range looks {
		name := names[need]
		if m.world != nil && need != coremachine.Hypervisor {
			look = func() bool { return !m.world.absent(name) }
		}
		p := coremachine.Provider{Name: name, Present: look(), Look: look}
		if slices.ContainsFunc(catalog.Apps, func(a catalog.App) bool { return a.Name == name }) {
			p.Item = name
		}
		out[need] = p
	}
	return out
}
