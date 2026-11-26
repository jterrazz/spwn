package calls

import (
	"context"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

func installing(pkg string, waits ...string) machinefile.Line {
	return machinefile.Line{Step: machinefile.Step{Package: pkg, Kind: catalog.KindApp, Method: "pkg", Waits: waits}, Action: machinefile.Install}
}

func outcomes(pkgs ...string) []apps.Outcome {
	out := make([]apps.Outcome, 0, len(pkgs))
	for _, p := range pkgs {
		out = append(out, apps.Outcome{Name: p, Result: catalog.StateOK})
	}
	return out
}

func TestAnApplyGivenABoundStopsBetweenPackagesOnItsClockAndAnswersWhatItDid(t *testing.T) {
	// invariant: a bound the caller gives is read on this machine's clock
	// before each package: one begun before it runs, those after it are
	// not run, and the run is cut, never an error.
	now := epoch()
	c := &calls{answer: outcomes("jq", "ripgrep", "node"), hook: func() { now = now.Add(10 * time.Minute) }}
	r := run{d: Deps{Calls: c, Ledger: &ledger{}, Now: func() time.Time { return now }}, in: ApplyIn{Within: 15 * time.Minute}, name: "web", ended: map[string]catalog.State{}, there: map[string]bool{}}
	ref, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	due, _ := ref.Deadline()
	got := r.carry(context.Background(), []machinefile.Line{installing("jq"), installing("ripgrep"), installing("node")})
	if len(got) != 3 || got[0].State != catalog.StateOK || got[1].State != catalog.StateOK || got[2].Reason != machinefile.NotRun {
		t.Fatalf("carried %+v; want jq and ripgrep installed, node not run", got)
	}
	if !r.cut {
		t.Fatal("the run ended at its bound but reads whole")
	}
	if off := c.until[0].Sub(due); off < 0 || off > time.Minute {
		t.Fatalf("an install ran until %v; want the bound given, %v", c.until[0], due)
	}
}

func TestAnApplyWithNoBoundGivenIsNeverCut(t *testing.T) {
	now := epoch()
	c := &calls{answer: outcomes("jq", "ripgrep"), hook: func() { now = now.Add(time.Hour) }}
	r := run{d: Deps{Calls: c, Ledger: &ledger{}, Now: func() time.Time { return now }}, name: "web", ended: map[string]catalog.State{}, there: map[string]bool{}}
	got := r.carry(context.Background(), []machinefile.Line{installing("jq"), installing("ripgrep")})
	if r.cut || got[1].State != catalog.StateOK {
		t.Fatalf("carried %+v, cut %v; want both installed, whole", got, r.cut)
	}
}

func TestAnApplyThatLeavesWhatAsksCarriesOutTheRest(t *testing.T) {
	// invariant: a package left since it would ask is named with why, what
	// requires it is not run, and every other package is carried out.
	c := &calls{answer: outcomes("jq", "gh-tool", "ripgrep")}
	r := run{d: Deps{Calls: c, Ledger: &ledger{}, Now: epoch}, in: ApplyIn{LeaveAsking: true}, name: "web", ended: map[string]catalog.State{}, there: map[string]bool{}, left: map[string]bool{"github-login": true}}
	got := r.carry(context.Background(), []machinefile.Line{installing("jq"), installing("github-login"), installing("gh-tool", "github-login"), installing("ripgrep")})
	if got[0].State != catalog.StateOK || got[3].State != catalog.StateOK {
		t.Fatalf("carried %+v; want jq and ripgrep installed", got)
	}
	if got[1].Reason != machinefile.WaitsForYou || got[1].Detail != machinefile.AsksInTerminal {
		t.Fatalf("github-login ended %q %q; want left, %q", got[1].Reason, got[1].Detail, machinefile.AsksInTerminal)
	}
	if got[2].Reason != machinefile.NotRun {
		t.Fatalf("gh-tool ended %q; want not run, since what it requires was left", got[2].Reason)
	}
	if len(c.until) != 2 {
		t.Fatalf("installed %d packages; want 2", len(c.until))
	}
}
