package ports

import (
	"slices"
	"strings"
)

func Of(f AllMachines, name string) AllMachines {
	// invariant: the ports of one machine, read through the Mac it is on: a
	// Mac's whole, or a VM's shares and listeners alone under its Mac, so a
	// server down still lists what it listened on. A Mac is named in full or
	// as the interface names it, without the `<user>-` its name starts with.
	out := AllMachines{Schema: f.Schema}
	for _, h := range f.Macs {
		if h.Name == name || strings.HasSuffix(h.Name, "-"+name) {
			out.Macs = []MacPorts{h}
			return out
		}
	}
	for _, h := range f.Macs {
		shared := slices.DeleteFunc(slices.Clone(h.Shared), func(s SharedPort) bool { return s.VM != name })
		listened := slices.DeleteFunc(slices.Clone(h.Ports), func(p Port) bool { return p.Machine.Name != name })
		if len(shared) == 0 && len(listened) == 0 {
			continue
		}
		h.Shared, h.Ports, h.Range, h.Listening = shared, listened, nil, nil
		out.Macs = []MacPorts{h}
		return out
	}
	return out
}
