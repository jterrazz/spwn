package machines

func ViewOf(name string, rec MacRecord, node Node, far MacView, answered bool) MacView {
	// invariant: another Mac as This Mac draws it: its own answer under the
	// name and user This Mac keeps for it, the tailnet's node when it has
	// one; one that did not answer, or answered on another release, as its
	// record last kept it, every remembered VM in an unknown state.
	if !answered || far.OtherRelease {
		silent := MacView{Name: name, User: rec.User, Node: node, Answered: rec.Answered, LAN: rec.LAN, Laptop: rec.Laptop, Asleep: rec.Asleep}
		if rec.Hardware != nil {
			silent.Hardware = *rec.Hardware
		}
		if answered {
			silent.StationVersion, silent.Reachable, silent.OtherRelease = far.StationVersion, true, true
		}
		for _, vm := range rec.VMs {
			// invariant: a VM remembered with no kind is left out, never
			// guessed; its Mac's next answer brings it back with its own.
			if vm.Kind == "" {
				continue
			}
			silent.VMs = append(silent.VMs, VM{
				Name: vm.Name, Mac: name, Kind: vm.Kind, Lifetime: vm.Lifetime, State: UnknownState,
				Resources: Resources{CPUs: vm.CPUs, MemoryBytes: vm.MemoryBytes},
			})
		}
		return silent
	}
	view := MacView{
		Name: name, User: rec.User, StationVersion: far.StationVersion, Reachable: true, LAN: far.LAN, Node: far.Node, Engine: far.Engine,
		Hardware: far.Hardware, Laptop: far.Laptop, Asleep: far.Asleep, Awake: far.Awake, SilentHypervisor: far.SilentHypervisor, SilentPackage: far.SilentPackage,
	}
	view.VMs = AnsweredVMs(name, far.VMs)
	if node.IP != "" || node.Online {
		view.Node = node
	}
	return view
}

func AnsweredVMs(mac string, vms []VM) []VM {
	// invariant: the VMs another Mac answered, each on that Mac; one
	// answered with no name is left out, as a broken or hostile binary
	// may send it: no page draws a row it cannot name.
	var out []VM
	for _, vm := range vms {
		if vm.Name == "" {
			continue
		}
		vm.Mac = mac
		out = append(out, vm)
	}
	return out
}
