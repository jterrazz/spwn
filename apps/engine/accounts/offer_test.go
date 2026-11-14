package accounts_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

func TestATargetThatCannotBePickedSaysWhy(t *testing.T) {
	signedInHere := accounts.Facts{State: accounts.ProgramSignedIn, How: accounts.HowOwn}
	for name, c := range map[string]struct {
		in   accounts.Candidate
		fit  accounts.TargetFit
		line string
	}{
		"a server":    {accounts.Candidate{ToKind: accounts.ToMachine, To: "infra", Where: accounts.Where{Kind: machine.Server}}, accounts.TargetNo, "A server keeps its own sign-ins"},
		"another Mac": {accounts.Candidate{ToKind: accounts.ToMachine, To: "ci-1", Where: accounts.Where{FarMac: "mini"}}, accounts.TargetNo, "It is on mini: grant from there"},
		"stopped":     {accounts.Candidate{ToKind: accounts.ToMachine, To: "api", Where: accounts.Where{Stopped: true}}, accounts.TargetNo, "Start the machine first"},
		"its own":     {accounts.Candidate{ToKind: accounts.ToMachine, To: "linux-1", Facts: signedInHere}, accounts.TargetNo, "It has its own sign-in"},
		"signed out":  {accounts.Candidate{ToKind: accounts.ToMachine, To: "linux-1"}, accounts.TargetFits, "Signed out here"},
		"granted":     {accounts.Candidate{ToKind: accounts.ToMachine, To: "linux-1", Granted: true, Facts: accounts.Facts{State: accounts.ProgramSignedIn, How: accounts.HowGranted}}, accounts.TargetFits, "Granted already; granting again starts a new term"},
		"an agent":    {accounts.Candidate{ToKind: accounts.ToAgent, To: "bot", Workspace: "linux-1"}, accounts.TargetFits, "Only its terminal sees it"},
	} {
		got := accounts.Offer(accounts.Candidates{AgentGrantable: true, List: []accounts.Candidate{c.in}}).Targets[0]
		if got.Fit != c.fit || got.Line != c.line {
			t.Errorf("%s: %q %q; want %q %q", name, got.Fit, got.Line, c.fit, c.line)
		}
	}
	got := accounts.Offer(accounts.Candidates{List: []accounts.Candidate{{ToKind: accounts.ToAgent, To: "bot"}}}).Targets[0]
	if got.Fit != accounts.TargetNo {
		t.Errorf("an agent took a password that is not fine-grained: %q", got.Fit)
	}
}

func TestEveryTermOfferedIsInBoundsAndOneIsTheDefault(t *testing.T) {
	defaults := map[accounts.GrantTo]int{}
	for _, term := range accounts.Terms() {
		if !accounts.TermInBounds(term.ToKind, term.For) {
			t.Errorf("%s offers %s, out of bounds", term.ToKind, term.Label)
		}
		if term.Default {
			defaults[term.ToKind]++
		}
	}
	if defaults[accounts.ToMachine] != 1 || defaults[accounts.ToAgent] != 1 {
		t.Errorf("defaults = %v; want one of each kind", defaults)
	}
	for _, term := range accounts.Terms() {
		if term.ToKind == accounts.ToAgent && term.For == 0 {
			t.Errorf("an agent is offered until removed")
		}
	}
}
