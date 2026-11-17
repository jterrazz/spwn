package machines

type Cleaned struct {
	// value: SPEC B5.9: every stopped ephemeral workspace that may go, with
	// what goes with it; the confirmation lists their losses, the clean-up
	// sheet draws them.
	Candidates []Candidate
	Deleted    []string
	// value: what stays: the Mac's ephemeral workspaces still running.
	Kept []string
}

type Candidate struct {
	Name string
	// value: what goes with it: its disk, the guest ports reserved for it,
	// and the grants it holds, by their label.
	DiskBytes int64
	Ports     []int
	Accounts  []string
}

func Cleanup(vms []VM) (gone, kept []VM) {
	// invariant: SPEC B5.9: a stopped ephemeral workspace may go; a
	// running one stays, and is named.
	for _, vm := range vms {
		switch {
		case vm.Lifetime != Ephemeral:
		case vm.State == Stopped:
			gone = append(gone, vm)
		case vm.State == Running:
			kept = append(kept, vm)
		}
	}
	return gone, kept
}

func CandidateOf(vm VM) Candidate {
	c := Candidate{Name: vm.Name, DiskBytes: vm.Resources.DiskBytes}
	if c.DiskBytes == 0 {
		c.DiskBytes = int64(vm.SizeGB) * 1e9
	}
	return c
}
