package machines

import "time"

type Cleaned struct {
	// value: SPEC B5.9: every throwaway that may go, with what goes with it;
	// the confirmation lists their losses, the clean-up sheet draws them.
	Candidates []Candidate
	Deleted    []string
	// value: what stays: the Mac's running VMs an agent still works in, and
	// its throwaways still running.
	Kept []string
}

type Candidate struct {
	Name string
	// value: its owner stopped it; otherwise its agent finished.
	Stopped bool
	// value: when its agent finished, how in its app's words, and for how
	// long it has sat idle since; zero for a stopped one with no agent.
	Ended  time.Time
	Reason string
	Idle   time.Duration
	// value: what goes with it: its disk, the guest ports reserved for it,
	// and the accounts shared with it, by their label.
	DiskBytes int64
	Ports     []int
	Accounts  []string
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

func CandidateOf(vm VM, now time.Time) Candidate {
	c := Candidate{Name: vm.Name, Stopped: vm.State == Stopped, DiskBytes: vm.Resources.DiskBytes}
	if c.DiskBytes == 0 {
		c.DiskBytes = int64(vm.SizeGB) * 1e9
	}
	if vm.Finished() {
		c.Ended, c.Reason = vm.Agent.Ended, vm.Agent.Reason
		if !c.Ended.IsZero() && now.After(c.Ended) {
			c.Idle = now.Sub(c.Ended)
		}
	}
	return c
}
