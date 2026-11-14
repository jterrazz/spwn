package accounts

import (
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

const (
	lineServer  = "A server keeps its own sign-ins"
	lineStopped = "Start the machine first"
)

type TargetFit string

const (
	TargetFits TargetFit = "fits"
	TargetNo   TargetFit = "no"
)

func (TargetFit) Enum() []string { return []string{string(TargetFits), string(TargetNo)} }

type Target struct {
	ToKind GrantTo
	To     string
	// value: the workspace an agent works in, empty for a machine.
	Workspace string `json:",omitempty"`
	Fit       TargetFit
	// invariant: a target that cannot be picked says why in Line; one that
	// can says what the grant meets there.
	Line string
}

type GrantOffer struct {
	Targets []Target
	// value: the sentence about the owner's presence the Mac asks for before a grant leaves it.
	Presence string
}

type Candidate struct {
	ToKind    GrantTo
	To        string
	Workspace string
	Where     Where
	// value: what the target's own read says of the entry's program.
	Facts Facts
	// value: whether this very password is granted there now.
	Granted bool
}

type Candidates struct {
	// value: whether the password may go to an agent.
	AgentGrantable bool
	List           []Candidate
}

const presenceSentence = "Touch ID will ask you to confirm before the password leaves this Mac"

func Offer(c Candidates) GrantOffer {
	targets := make([]Target, 0, len(c.List))
	for _, k := range c.List {
		targets = append(targets, targetOf(k, c.AgentGrantable))
	}
	return GrantOffer{Targets: targets, Presence: presenceSentence}
}

func targetOf(k Candidate, agentGrantable bool) Target {
	t := Target{ToKind: k.ToKind, To: k.To, Workspace: k.Workspace, Fit: TargetNo}
	if k.ToKind == ToAgent {
		if !agentGrantable {
			t.Line = "Only a fine-grained token goes to an agent"
			return t
		}
		t.Fit, t.Line = TargetFits, SentenceAgentReach.Text()
		return t
	}
	switch {
	case k.Where.Kind == machine.Server:
		t.Line = lineServer
	case k.Where.FarMac != "":
		t.Line = "It is on " + k.Where.FarMac + ": grant from there"
	case k.Where.Stopped:
		t.Line = lineStopped
	case k.Facts.State == ProgramSignedIn && k.Facts.How == HowOwn:
		t.Line = "It has its own sign-in"
	case k.Granted:
		t.Fit, t.Line = TargetFits, "Granted already; granting again replaces it"
	case k.Facts.State == ProgramBroken:
		t.Fit, t.Line = TargetFits, "Its sign-in needs signing in again"
	default:
		t.Fit, t.Line = TargetFits, "Signed out here"
	}
	return t
}
