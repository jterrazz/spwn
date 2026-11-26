package calls

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	machinefilecalls "github.com/jterrazz/studio-station/apps/engine/machinefile/calls"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

func applying(answer machinefile.Applied, err error, until *time.Time) *Machinist {
	apply := func(ctx context.Context, _, _ string) (machinefile.Applied, error) {
		if until != nil {
			*until, _ = ctx.Deadline()
		}
		return answer, err
	}
	return &Machinist{h: keepingMac{kept: map[string]string{}}, o: Options{Apply: apply}}
}

func done(pkg string, action machinefile.Action, state catalog.State, reason machinefile.Reason, detail string) machinefile.Done {
	return machinefile.Done{Line: machinefile.Line{Step: machinefile.Step{Package: pkg}, Action: action}, State: state, Reason: reason, Detail: detail}
}

func TestACreatesApplyIsBoundedByThePackagesShare(t *testing.T) {
	// invariant: a create whose guest never answers still ends: its apply is
	// given the packages' share of a phase's bound, never more.
	var until time.Time
	ref, cancel := context.WithTimeout(context.Background(), catalog.PackagesBound)
	defer cancel()
	due, _ := ref.Deadline()
	applying(machinefile.Applied{}, nil, &until).applyAll(context.Background(), "web", "studio-linux", &model.FirstStart{})
	if off := until.Sub(due); off < 0 || off > time.Minute {
		t.Fatalf("the apply had %v; want %v", until.Sub(due)+catalog.PackagesBound, catalog.PackagesBound)
	}
}

func TestACreatesApplyCountsWhatItDidAndNamesWhatItLeft(t *testing.T) {
	answer := machinefile.Applied{Done: []machinefile.Done{
		done("fd", machinefile.Keep, catalog.StateOK, "", ""),
		done("jq", machinefile.Install, catalog.StateOK, "", ""),
		done("hushlogin", machinefile.Set, catalog.StateOK, "", ""),
		done("ripgrep", machinefile.Install, catalog.StateBroken, "", "the install failed\nits tail"),
		done("node", machinefile.Install, catalog.StatePending, machinefile.Interrupted, ""),
	}}
	b := &model.FirstStart{}
	applying(answer, errors.New("1 of 3 failed"), nil).applyAll(context.Background(), "web", "studio-linux", b)
	wantApplied := []model.FirstStartApp{{App: "jq"}, {App: "hushlogin", Setting: true}}
	wantLeft := []model.FirstStartApp{{App: "ripgrep", Reason: "the install failed"}, {App: "node", Reason: model.NotFinished}}
	if !slices.Equal(b.Applied, wantApplied) || !slices.Equal(b.Unfinished, wantLeft) || b.Failed != "" || b.Unapplied != "" {
		t.Fatalf("got applied %v, unfinished %v, failed %q, unapplied %q; want %v and %v alone", b.Applied, b.Unfinished, b.Failed, b.Unapplied, wantApplied, wantLeft)
	}
}

func TestACreatesApplyThatWouldAskLeavesEachAskedPackageUnfinished(t *testing.T) {
	refused := fail.New(fail.NeedsTerminal, machinefilecalls.NeedsTerminal, "xcode, ohmyzsh", nil)
	b := &model.FirstStart{}
	applying(machinefile.Applied{}, refused, nil).applyAll(context.Background(), "web", "studio-linux", b)
	want := []model.FirstStartApp{{App: "xcode", Reason: model.AsksYou}, {App: "ohmyzsh", Reason: model.AsksYou}}
	if !slices.Equal(b.Unfinished, want) || b.Failed != "" || b.Unapplied != "" {
		t.Fatalf("got unfinished %v, failed %q, unapplied %q; want %v alone", b.Unfinished, b.Failed, b.Unapplied, want)
	}
}

func TestACreatesApplyThatAnswersForNoPackageKeepsTheWorkspace(t *testing.T) {
	b := &model.FirstStart{}
	applying(machinefile.Applied{}, errors.New("the guest did not answer\nwithin its bound"), nil).applyAll(context.Background(), "web", "studio-linux", b)
	if b.Unapplied != "the guest did not answer" || b.Failed != "" || len(b.Unfinished) != 0 {
		t.Fatalf("got unapplied %q, failed %q, unfinished %v; want why in unapplied alone", b.Unapplied, b.Failed, b.Unfinished)
	}
}
