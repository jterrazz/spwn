package calls

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

// value: what a create's apply was handed: the bound the guest is given,
// and when the Mac stops waiting for its answer.
type handed struct {
	within time.Duration
	until  time.Time
}

func applying(answer machinefile.Applied, err error, got *handed) *Machinist {
	apply := func(ctx context.Context, _, _ string, within time.Duration) (machinefile.Applied, error) {
		if got != nil {
			got.within = within
			got.until, _ = ctx.Deadline()
		}
		return answer, err
	}
	return &Machinist{h: keepingMac{kept: map[string]string{}}, o: Options{Apply: apply}}
}

func done(pkg string, action machinefile.Action, state catalog.State, reason machinefile.Reason, detail string) machinefile.Done {
	return machinefile.Done{Line: machinefile.Line{Step: machinefile.Step{Package: pkg}, Action: action}, State: state, Reason: reason, Detail: detail}
}

func TestACreatesApplyBoundsTheGuestsWorkAndWaitsAGraceBeyondIt(t *testing.T) {
	// invariant: the guest is given the packages' share of a phase's bound,
	// and the Mac waits that, the cap of the package running at it and a
	// minute more, never less, so the guest answers first, and never more,
	// so a guest that never answers still lets the create end.
	if applyGrace < catalog.InFlightCap+time.Minute {
		t.Fatalf("the Mac's grace %v gives up before the guest's cap %v and its minute", applyGrace, catalog.InFlightCap)
	}
	var got handed
	ref, cancel := context.WithTimeout(context.Background(), catalog.PackagesBound+applyGrace)
	defer cancel()
	due, _ := ref.Deadline()
	applying(machinefile.Applied{}, nil, &got).applyAll(context.Background(), "web", "studio-linux", &model.FirstStart{})
	if got.within != catalog.PackagesBound {
		t.Fatalf("the guest was given %v; want %v", got.within, catalog.PackagesBound)
	}
	if off := got.until.Sub(due); off < 0 || off > time.Minute {
		t.Fatalf("the Mac waited %v; want %v", got.until.Sub(due)+catalog.PackagesBound+applyGrace, catalog.PackagesBound+applyGrace)
	}
}

func TestACreatesApplyCountsWhatItDidAndNamesWhatItLeft(t *testing.T) {
	answer := machinefile.Applied{Done: []machinefile.Done{
		done("fd", machinefile.Keep, catalog.StateOK, "", ""),
		done("jq", machinefile.Install, catalog.StateOK, "", ""),
		done("hushlogin", machinefile.Set, catalog.StateOK, "", ""),
		done("ripgrep", machinefile.Install, catalog.StateBroken, "", "the install failed\nits tail"),
		done("github-login", machinefile.Install, catalog.StateAbsent, machinefile.WaitsForYou, machinefile.AsksInTerminal),
		done("node", machinefile.Install, catalog.StateBroken, machinefile.Interrupted, ""),
		done("tree", machinefile.Install, catalog.StateAbsent, machinefile.NotRun, ""),
	}}
	b := &model.FirstStart{}
	applying(answer, nil, nil).applyAll(context.Background(), "web", "studio-linux", b)
	wantApplied := []model.FirstStartApp{{App: "jq"}, {App: "hushlogin", Setting: true}}
	wantLeft := []model.FirstStartApp{
		{App: "ripgrep", Reason: "the install failed"},
		{App: "github-login", Reason: machinefile.AsksInTerminal},
		{App: "node", Reason: model.StillInstalling},
		{App: "tree", Reason: model.NotFinished},
	}
	if !slices.Equal(b.Applied, wantApplied) || !slices.Equal(b.Unfinished, wantLeft) || b.Failed != "" || b.Unapplied != "" {
		t.Fatalf("got applied %v, unfinished %v, failed %q, unapplied %q; want %v and %v alone", b.Applied, b.Unfinished, b.Failed, b.Unapplied, wantApplied, wantLeft)
	}
}

func TestACreatesApplyThatAnswersForNoPackageKeepsTheWorkspace(t *testing.T) {
	b := &model.FirstStart{}
	answer := machinefile.Applied{Done: []machinefile.Done{done("jq", machinefile.Install, catalog.StateOK, "", "")}}
	applying(answer, errors.New("the guest did not answer\nwithin its bound"), nil).applyAll(context.Background(), "web", "studio-linux", b)
	if b.Unapplied != "the guest did not answer" || b.Failed != "" || len(b.Unfinished) != 0 || len(b.Applied) != 0 {
		t.Fatalf("got unapplied %q, failed %q, applied %v, unfinished %v; want why in unapplied alone", b.Unapplied, b.Failed, b.Applied, b.Unfinished)
	}
}

func TestACreatesApplyPastItsGraceSaysItRanOutOfTime(t *testing.T) {
	b := &model.FirstStart{}
	applying(machinefile.Applied{}, context.DeadlineExceeded, nil).applyAll(context.Background(), "web", "studio-linux", b)
	if b.Unapplied != model.OutOfTime {
		t.Fatalf("got unapplied %q; want %q, never the context's own words", b.Unapplied, model.OutOfTime)
	}
}

// value: an engine whose guest runs no line.
type deafEngine struct{ Engine }

func (deafEngine) GuestExec(context.Context, string, string, string) (GuestRan, error) {
	return GuestRan{}, errors.New("the guest did not answer")
}

func TestACreateWhoseMachinefileWasNotCopiedSaysItsApplyDidNotRun(t *testing.T) {
	applied := false
	apply := func(context.Context, string, string, time.Duration) (machinefile.Applied, error) {
		applied = true
		return machinefile.Applied{}, nil
	}
	s := &Machinist{engineAPI: deafEngine{}, h: keepingMac{kept: map[string]string{}}, o: Options{Apply: apply}}
	b := s.bootPhase(context.Background(), "web", "v0.21.0", model.Template{Name: "studio-linux", Files: []model.TemplateFile{{Path: "machinefiles/studio-linux.machine.yaml"}}})
	if applied || b.Failed == "" || b.Unapplied != model.NeverRan {
		t.Fatalf("applied %v, failed %q, unapplied %q; want no apply, the failure, and %q", applied, b.Failed, b.Unapplied, model.NeverRan)
	}
}

func TestACreateStoppedDuringItsApplySaysSo(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	apply := func(ctx context.Context, _, _ string, _ time.Duration) (machinefile.Applied, error) {
		stop()
		<-ctx.Done()
		return machinefile.Applied{}, ctx.Err()
	}
	b := &model.FirstStart{}
	(&Machinist{h: keepingMac{kept: map[string]string{}}, o: Options{Apply: apply}}).applyAll(ctx, "web", "studio-linux", b)
	if b.Unapplied != model.ApplyStopped {
		t.Fatalf("got unapplied %q; want %q", b.Unapplied, model.ApplyStopped)
	}
}
