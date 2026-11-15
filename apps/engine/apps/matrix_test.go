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
	}, nil)
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
