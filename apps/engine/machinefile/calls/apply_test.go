package calls

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	appscalls "github.com/jterrazz/studio-station/apps/engine/apps/calls"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

// value: this machine's calls as a run of apply reaches them: an install
// answers what answer says and then hook runs, an uninstall fails for the
// items of broken, and every call is noted.
type calls struct {
	answer []apps.Outcome
	hook   func()
	broken []string
	asked  []string
	// value: the deadline of each install, zero for none.
	until []time.Time
}

func (c *calls) Invoke(ctx context.Context, _ machine.Ref, name string, in, out any, _ call.Progress) error {
	c.asked = append(c.asked, name)
	switch name {
	case appscalls.Install.Name:
		until, _ := ctx.Deadline()
		c.until = append(c.until, until)
		give(out, c.answer)
		if c.hook != nil {
			c.hook()
		}
	case appscalls.Remove.Name:
		req, _ := in.(appscalls.RemoveIn)
		pkg := req.Names[0]
		c.asked[len(c.asked)-1] += " " + pkg
		if slices.Contains(c.broken, pkg) {
			return errors.New(pkg + " did not uninstall")
		}
		give(out, []apps.Outcome{{Name: pkg, Result: catalog.StateOK}})
	}
	return nil
}

func give(out any, o []apps.Outcome) {
	if p, ok := out.(*[]apps.Outcome); ok {
		*p = o
	}
}

func (c *calls) Watch(_ context.Context, _ machine.Ref, name string, _ any, _ call.Progress) (<-chan call.Update[any], error) {
	c.asked = append(c.asked, name)
	updates := make(chan call.Update[any])
	close(updates)
	return updates, nil
}

func epoch() time.Time { return time.Unix(0, 0) }

type ledger struct{ deleted []string }

func (*ledger) List(context.Context) ([]machinefile.Installed, error) { return nil, nil }
func (*ledger) Put(context.Context, machinefile.Installed) error      { return nil }
func (*ledger) Night(context.Context) (machinefile.Night, bool, error) {
	return machinefile.Night{}, false, nil
}
func (*ledger) Write(context.Context, machinefile.Night) error { return nil }
func (l *ledger) Delete(_ context.Context, pkg string) error {
	l.deleted = append(l.deleted, pkg)
	return nil
}
func (*ledger) Blocks(context.Context) ([]machinefile.Written, error) { return nil, nil }
func (*ledger) PutBlock(context.Context, machinefile.Written) error   { return nil }
func (*ledger) DeleteBlock(context.Context, string) error             { return nil }

func TestAnInstallThatAnsweredBeforeTheInterruptIsNotReadAgain(t *testing.T) {
	// invariant: the read past an interrupt is for an item whose install
	// handed back nothing; one that answered already installed is kept as
	// it said.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &calls{answer: []apps.Outcome{{Name: "tool", Result: catalog.StateOK, Detail: apps.AlreadyInstalled}}, hook: cancel}
	r := run{d: Deps{Calls: c, Ledger: &ledger{}, Now: epoch}, name: "laptop", ended: map[string]catalog.State{}}
	l := machinefile.Line{Step: machinefile.Step{Package: "tool", Kind: catalog.KindApp, Method: "pkg"}, Action: machinefile.Install}
	done := r.install(ctx, l, machinefile.Done{Line: l})
	if slices.Contains(c.asked, appscalls.List.Name) {
		t.Errorf("asked %v; want no read past the interrupt", c.asked)
	}
	if done.State != catalog.StateOK || done.Detail != apps.AlreadyInstalled || done.Installed {
		t.Errorf("ended %s %q installed %v; want ok, already installed, not recorded", done.State, done.Detail, done.Installed)
	}
}

func TestAnExtraThatDidNotUninstallKeepsWhatItLeansOn(t *testing.T) {
	// invariant: an extra whose uninstall failed is still there, so the
	// extras it leans on are kept with their record.
	c := &calls{broken: []string{"browsers"}}
	l := &ledger{}
	r := run{d: Deps{Calls: c, Ledger: l, Now: epoch}, name: "laptop", ended: map[string]catalog.State{}}
	extra := func(pkg string, waits ...string) machinefile.ExtraLine {
		return machinefile.ExtraLine{Line: machinefile.Line{
			Step:   machinefile.Step{Package: pkg, Kind: catalog.KindApp, Method: "pkg", Waits: waits},
			Seen:   machinefile.Seen{Condition: machinefile.Present, Method: "pkg"},
			Action: machinefile.Remove,
		}}
	}
	out, _ := r.remove(context.Background(), machinefile.Plan{Extra: []machinefile.ExtraLine{extra("browsers", "runtime"), extra("runtime", "base"), extra("base"), extra("other")}})
	if want := []string{appscalls.Remove.Name + " browsers", appscalls.Remove.Name + " other"}; !slices.Equal(c.asked, want) {
		t.Errorf("uninstalled %v; want %v", c.asked, want)
	}
	if want := []string{"other"}; !slices.Equal(l.deleted, want) {
		t.Errorf("forgot %v; want %v", l.deleted, want)
	}
	for _, d := range out[1:3] {
		if d.Detail == "" {
			t.Errorf("%s kept with no words", d.Line.Step.Package)
		}
	}
}

func TestAPhaseInAVMHoldsItsPackagesToTheirShare(t *testing.T) {
	// invariant: inside a VM's phase (image, boot) the items run within
	// catalog.ItemsBound, so the owner's lines keep catalog.RunBound of the
	// phase's bound; apply's own phase bounds them by nothing.
	l := machinefile.Line{Step: machinefile.Step{Package: "tool", Kind: catalog.KindApp, Method: "pkg"}, Action: machinefile.Install}
	for _, c := range []struct {
		phase catalog.Phase
		share time.Duration
	}{{catalog.PhaseBoot, catalog.PackagesBound}, {catalog.PhaseImage, catalog.PackagesBound}, {catalog.PhaseApply, 0}} {
		t.Run(string(c.phase), func(t *testing.T) {
			fake := &calls{answer: []apps.Outcome{{Name: "tool", Result: catalog.StateOK}}}
			r := run{d: Deps{Calls: fake, Ledger: &ledger{}, Now: epoch}, in: ApplyIn{Phase: c.phase}, name: "ws", ended: map[string]catalog.State{}, there: map[string]bool{}}
			ref, cancel := context.WithTimeout(context.Background(), c.share)
			defer cancel()
			due, _ := ref.Deadline()
			r.carry(context.Background(), []machinefile.Line{l})
			if len(fake.until) != 1 {
				t.Fatalf("installed %d times; want once", len(fake.until))
			}
			got := fake.until[0]
			if c.share == 0 {
				if !got.IsZero() {
					t.Fatalf("the install ran until %v; want no bound", got)
				}
				return
			}
			if off := got.Sub(due); off < 0 || off > time.Minute {
				t.Fatalf("the install had %v; want the items' share, %v", off+c.share, c.share)
			}
		})
	}
}
