package approvals

import "github.com/jterrazz/studio-station/apps/engine/machines"

type State string

// value: how macOS stands with one approval, a consent macOS gives Spwn,
// which Spwn can ask for and read but never give itself (ADR-105).
const (
	// value: macOS lets Spwn do it.
	Approved State = "approved"
	// value: macOS refuses it, and only System Settings changes that now.
	Refused State = "refused"
	// value: a refusal that is not yet macOS's verdict.
	Checking State = "checking"
	// value: a read that cannot tell: off a Mac, or one that did not end.
	Unknown State = "unknown"
)

func (State) Enum() []string {
	return []string{string(Approved), string(Refused), string(Checking), string(Unknown)}
}

func (s State) Word() string {
	switch s {
	case Approved:
		return "approved"
	case Refused:
		return "refused"
	case Checking:
		return "checking"
	case Unknown:
		return "unknown"
	}
	return string(s)
}

func StateOf(read machines.LocalNetwork) State {
	// value: the state of the engine's Local Network read as its hello
	// carries it; a read the hello leaves empty cannot tell.
	switch read {
	case machines.LocalNetworkApproved:
		return Approved
	case machines.LocalNetworkNotApproved:
		return Refused
	case machines.LocalNetworkChecking:
		return Checking
	}
	return Unknown
}

func (s State) Hello() machines.LocalNetwork {
	// value: the Local Network read a state is carried as in the engine's
	// hello, in the hello's own words.
	switch s {
	case Approved:
		return machines.LocalNetworkApproved
	case Refused:
		return machines.LocalNetworkNotApproved
	case Checking:
		return machines.LocalNetworkChecking
	case Unknown:
	}
	return ""
}
