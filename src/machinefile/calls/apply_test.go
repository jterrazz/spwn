package calls

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/apps"
	appscalls "github.com/jterrazz/studio-station/src/apps/calls"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/machinefile"
)

// value: this machine's calls as a run of apply reaches them: an install
// answers what answer says and then hook runs, an uninstall fails for the
// items of broken, and every call is noted.
type calls struct {
	answer []apps.Outcome
	hook   func()
	broken []string
	asked  []string
}

func (c *calls) Invoke(_ context.Context, _ machine.Ref, name string, in, out any, _ call.Progress) error {
	c.asked = append(c.asked, name)
	switch name {
	case appscalls.Install.Name:
		give(out, c.answer)
		if c.hook != nil {
			c.hook()
		}
	case appscalls.Remove.Name:
		req, _ := in.(appscalls.RemoveIn)
		item := req.Names[0]
		c.asked[len(c.asked)-1] += " " + item
		if slices.Contains(c.broken, item) {
			return errors.New(item + " did not uninstall")
		}
		give(out, []apps.Outcome{{Name: item, Result: catalog.StateOK}})
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
func (l *ledger) Delete(_ context.Context, item string) error {
	l.deleted = append(l.deleted, item)
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
	l := machinefile.Line{Step: machinefile.Step{Item: "tool", Kind: catalog.KindApp, Method: "pkg"}, Action: machinefile.Install}
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
	extra := func(item string, waits ...string) machinefile.ExtraLine {
		return machinefile.ExtraLine{Line: machinefile.Line{
			Step:   machinefile.Step{Item: item, Kind: catalog.KindApp, Method: "pkg", Waits: waits},
			Seen:   machinefile.Seen{Condition: machinefile.Present, Method: "pkg"},
			Action: machinefile.Remove,
		}}
	}
	out := r.remove(context.Background(), []machinefile.ExtraLine{extra("browsers", "runtime"), extra("runtime", "base"), extra("base"), extra("other")})
	if want := []string{appscalls.Remove.Name + " browsers", appscalls.Remove.Name + " other"}; !slices.Equal(c.asked, want) {
		t.Errorf("uninstalled %v; want %v", c.asked, want)
	}
	if want := []string{"other"}; !slices.Equal(l.deleted, want) {
		t.Errorf("forgot %v; want %v", l.deleted, want)
	}
	for _, d := range out[1:3] {
		if d.Detail == "" {
			t.Errorf("%s kept with no words", d.Line.Step.Item)
		}
	}
}
