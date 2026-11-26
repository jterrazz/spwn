package approvals

import "time"

// value: how long one Local Network read may take; how long a refusal past
// the engine's start reads checking; how often a checking read is read
// again, and a refused one.
const (
	LocalNetworkBound    = time.Second
	LocalNetworkSettling = 90 * time.Second
	LocalNetworkRetry    = 2 * time.Second
	LocalNetworkEvery    = 30 * time.Second
)

func Settled(read State, sinceStart time.Duration) State {
	// invariant: macOS refuses the first datagram of a bundle it has just
	// seen replaced, approved or not, and lets later ones through once it
	// has caught up, within about a minute. So a refusal in the engine's
	// first 90 s reads checking, never refused: only a refusal past them is
	// the Mac's verdict.
	if read == Refused && sinceStart < LocalNetworkSettling {
		return Checking
	}
	return read
}

func Due(read State, sinceRead time.Duration) bool {
	// invariant: a checking read is read again every 2 s, a refused one
	// every 30 s, so the owner's click is seen without a restart; once
	// approved, or where it cannot tell, it is read no more.
	switch read {
	case Checking:
		return sinceRead >= LocalNetworkRetry
	case Refused:
		return sinceRead >= LocalNetworkEvery
	case Approved, Unknown:
	}
	return false
}
