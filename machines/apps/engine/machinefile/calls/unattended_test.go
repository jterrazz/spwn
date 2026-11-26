package calls

import (
	"context"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	appscalls "github.com/jterrazz/studio-station/apps/engine/apps/calls"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
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

func TestAnApplyGivenABoundStartsNoPackagePastItOnItsClockAndAnswersWhatItDid(t *testing.T) {
	// invariant: a bound the caller gives is read on this machine's clock
	// before each package: one begun before it runs, ending past it, those
	// after it are not run, and the run is cut, never an error; each
	// install may run until the bound and the cap of the one in flight.
	now := epoch()
	c := &calls{answer: outcomes("jq", "ripgrep", "node"), hook: func() { now = now.Add(10 * time.Minute) }}
	r := run{d: Deps{Calls: c, Ledger: &ledger{}, Now: func() time.Time { return now }}, in: ApplyIn{Within: 15 * time.Minute}, name: "web", ended: map[string]catalog.State{}, there: map[string]bool{}, inFlight: catalog.InFlightCap}
	ref, cancel := context.WithTimeout(context.Background(), 15*time.Minute+catalog.InFlightCap)
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
		t.Fatalf("an install ran until %v; want the bound given and the cap, %v", c.until[0], due)
	}
}

// value: an install that never ends on its own, as apt behind a held
// lock: it moves the run's clock past its bound, then waits for its
// context's end.
type stuckCalls struct {
	*calls
	moved func()
}

func (s stuckCalls) Invoke(ctx context.Context, target machine.Ref, name string, in, out any, p call.Progress) error {
	if name == appscalls.Install.Name {
		s.moved()
		<-ctx.Done()
		return ctx.Err()
	}
	return s.calls.Invoke(ctx, target, name, in, out, p)
}

func TestAPackageInFlightAtTheBoundEndsWholeWithinItsCap(t *testing.T) {
	// invariant: the package running at the bound is never stopped there:
	// its install may run the cap past the bound, it ends installed, and
	// the next is not started.
	now := epoch()
	c := &calls{answer: outcomes("jq"), hook: func() { now = now.Add(10 * time.Minute) }}
	r := run{d: Deps{Calls: c, Ledger: &ledger{}, Now: func() time.Time { return now }}, in: ApplyIn{Within: 5 * time.Minute}, name: "web", ended: map[string]catalog.State{}, there: map[string]bool{}, inFlight: catalog.InFlightCap}
	bound, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	at, _ := bound.Deadline()
	got := r.carry(context.Background(), []machinefile.Line{installing("jq"), installing("ripgrep")})
	if got[0].State != catalog.StateOK || got[0].Reason != "" || got[1].Reason != machinefile.NotRun || !r.cut {
		t.Fatalf("carried %+v, cut %v; want jq installed past the bound, ripgrep not started, the run cut", got, r.cut)
	}
	if past := c.until[0].Sub(at); past < catalog.InFlightCap-time.Minute {
		t.Fatalf("jq's install was given %v past the bound; want its cap, %v", past, catalog.InFlightCap)
	}
}

func TestAPackageStillInstallingAtItsCapEndsThereAndSaysSo(t *testing.T) {
	// invariant: an install past the bound and its cap is ended (its
	// context, which ends what it started), and its line says it was
	// interrupted, never installed or failed.
	now := epoch()
	c := stuckCalls{calls: &calls{answer: outcomes("jq")}, moved: func() { now = now.Add(time.Hour) }}
	r := run{d: Deps{Calls: c, Ledger: &ledger{}, Now: func() time.Time { return now }}, in: ApplyIn{Within: 50 * time.Millisecond}, name: "web", ended: map[string]catalog.State{}, there: map[string]bool{}, inFlight: 100 * time.Millisecond}
	got := r.carry(context.Background(), []machinefile.Line{installing("jq"), installing("ripgrep")})
	if got[0].Reason != machinefile.Interrupted || got[1].Reason != machinefile.NotRun || !r.cut {
		t.Fatalf("carried %+v, cut %v; want jq interrupted at its cap, ripgrep not started, the run cut", got, r.cut)
	}
}

func TestAnApplyWithNoBoundGivenIsNeverCut(t *testing.T) {
	// invariant: Within zero or less is no bound, as ApplyIn says.
	for _, within := range []time.Duration{0, -time.Minute} {
		now := epoch()
		c := &calls{answer: outcomes("jq", "ripgrep"), hook: func() { now = now.Add(time.Hour) }}
		r := run{d: Deps{Calls: c, Ledger: &ledger{}, Now: func() time.Time { return now }}, in: ApplyIn{Within: within}, name: "web", ended: map[string]catalog.State{}, there: map[string]bool{}, inFlight: catalog.InFlightCap}
		got := r.carry(context.Background(), []machinefile.Line{installing("jq"), installing("ripgrep")})
		if r.cut || got[1].State != catalog.StateOK || !c.until[0].IsZero() {
			t.Fatalf("within %v carried %+v, cut %v, install deadline %v; want both installed, whole, unbounded", within, got, r.cut, c.until[0])
		}
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
