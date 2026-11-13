package fake

import "github.com/jterrazz/studio-station/apps/engine/world"

type inVM struct {
	// invariant: one VM of the world, found on the Mac that holds it; a VM
	// no Mac holds is made empty on This Mac the first time it is asked.
	w  *world.World
	vm string
}

func (x inVM) agents() *world.Agents {
	mac := x.w.Holder(x.vm)
	if mac == "" {
		mac = x.w.ThisMac
	}
	m := x.w.Mac(mac)
	v := m.VMs[x.vm]
	if v == nil {
		v = &world.VM{}
		m.VMs[x.vm] = v
	}
	return v.Agentic()
}

func alone() inVM { return inVM{w: &world.World{ThisMac: "this-mac"}, vm: "linux-1"} }
