package apps_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func TestACellHoldsTheAppWhenItsStateIsInstalledInAnyForm(t *testing.T) {
	// invariant: Holds is State.Installed of the cell's own state, so the
	// machines a row counts holding are the machines Carry counts.
	report := func(state catalog.State, managed bool) apps.Report {
		return apps.Report{App: catalog.App{Name: "node"}, State: state, Managed: managed}
	}
	cols := make([]apps.Column, 4)
	m := apps.MatrixOf(cols, []apps.Reading{
		{Read: true, Reports: []apps.Report{report(catalog.StateOK, true)}},
		{Read: true, Reports: []apps.Report{report(catalog.StateOK, false)}},
		{Read: true, Reports: []apps.Report{report(catalog.StateAbsent, false)}},
		{},
	}, nil, nil)
	if len(m.Rows) != 1 {
		t.Fatalf("%d rows", len(m.Rows))
	}
	holds := 0
	for i, want := range []bool{true, true, false, false} {
		if got := m.Rows[0].Cells[i].Holds; got != want {
			t.Errorf("cell %d (%s): holds %v, want %v", i, m.Rows[0].Cells[i].State, got, want)
		}
		if m.Rows[0].Cells[i].Holds {
			holds++
		}
	}
	if holds != m.Rows[0].Carry {
		t.Errorf("%d cells hold it, Carry says %d", holds, m.Rows[0].Carry)
	}
}

func TestTheShelvesSayWhatMachinesCarryAndWhatTheCatalogHolds(t *testing.T) {
	skill := func(name string) apps.Report {
		return apps.Report{App: catalog.App{Name: name, AppKind: catalog.AppKindSkill}, State: catalog.StateOK}
	}
	app := apps.Report{App: catalog.App{Name: "node", AppKind: catalog.AppKindApp}, State: catalog.StateOK}
	known := []catalog.App{app.App, skill("a").App, skill("b").App, skill("c").App}
	cols := make([]apps.Column, 1)
	many := apps.MatrixOf(cols, []apps.Reading{{Read: true, Reports: []apps.Report{app, skill("a"), skill("b")}}}, nil, known)
	if got, want := many.Skills.Mine.Sentence, "2 skills are on your machines"; got != want {
		t.Errorf("mine %q, want %q", got, want)
	}
	if got, want := many.Skills.Catalog.Sentence, "3 skills are in the catalog"; got != want {
		t.Errorf("catalog %q, want %q", got, want)
	}
	if got := many.Verdict.Mine; got.Sentence != apps.EveryAppFollowsItsRule || got.Tone != "ok" {
		t.Errorf("apps %+v", got)
	}
	one := apps.MatrixOf(cols, []apps.Reading{{Read: true, Reports: []apps.Report{skill("a")}}}, nil, known[:2])
	if got, want := one.Skills.Mine.Sentence, "1 skill is on your machines"; got != want {
		t.Errorf("one %q, want %q", got, want)
	}
	if got, want := one.Skills.Catalog.Sentence, "1 skill is in the catalog"; got != want {
		t.Errorf("one catalog %q, want %q", got, want)
	}
}
