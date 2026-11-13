package agents

func Speaker(as []Agent) (Agent, bool) {
	// invariant: the one agent a VM is said by: one waiting on the owner
	// first, then one at work, then the one that finished last.
	var best Agent
	found := false
	rank := func(a Agent) int {
		switch a.State {
		case Waiting:
			return 0
		case Finished:
			return 2
		default:
			return 1
		}
	}
	for _, a := range as {
		switch {
		case !found, rank(a) < rank(best), rank(a) == 2 && rank(best) == 2 && a.Ended.After(best.Ended):
			best, found = a, true
		}
	}
	return best, found
}
