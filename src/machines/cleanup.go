package machines

type Cleaned struct {
	Deleted []string
	// value: what stays: the Mac's running VMs an agent still works in, and
	// its throwaways still running.
	Kept []string
}

func Cleanup(vms []VM) (gone, kept []VM) {
	// invariant: SPEC B5.9: a throwaway that is stopped, or whose agent
	// finished, may go; a running VM an agent still works in, and a
	// throwaway still running, stay, and are named.
	for _, vm := range vms {
		switch {
		case vm.Lifetime == Throwaway && (vm.State == Stopped || vm.Finished()):
			gone = append(gone, vm)
		case vm.State == Running && (vm.Lifetime == Throwaway || vm.Agent != nil):
			kept = append(kept, vm)
		}
	}
	return gone, kept
}
