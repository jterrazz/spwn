package settings_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/settings"
)

func TestTheSettingsShelfCountsWhatSomeMachineHoldsAndLeavesOutChecks(t *testing.T) {
	set := func(name string, group settings.Group, state settings.State) settings.Row {
		return settings.Row{Name: name, Group: group, State: state}
	}
	check := settings.Row{Name: "firewall", Group: settings.Checked, Checked: settings.NeedsFix}
	cols := []settings.Column{
		{Rows: []settings.Row{set("alpha", settings.SetByStation, settings.StateApplied), set("zsh", settings.SetByStation, settings.StateNotApplied), check}},
		{Rows: []settings.Row{set("alpha", settings.SetByStation, settings.StateNotApplied), set("zsh", settings.SetByStation, settings.StateChangedSince), check}},
	}
	m := settings.MatrixOf(cols, nil, 7)
	if got, want := m.Verdict.Mine.Sentence, "2 settings are on your machines"; got != want {
		t.Errorf("mine %q, want %q", got, want)
	}
	if got, want := m.Verdict.Catalog.Sentence, "7 settings are in the catalog"; got != want {
		t.Errorf("catalog %q, want %q", got, want)
	}
	if !m.Rows[0].Cells[0].Holds || m.Rows[0].Cells[1].Holds {
		t.Errorf("alpha cells hold %v, %v", m.Rows[0].Cells[0].Holds, m.Rows[0].Cells[1].Holds)
	}
	one := settings.MatrixOf(cols[:1], nil, 1)
	if got, want := one.Verdict.Mine.Sentence, "1 setting is on your machines"; got != want {
		t.Errorf("one %q, want %q", got, want)
	}
	if got, want := one.Verdict.Catalog.Sentence, "1 setting is in the catalog"; got != want {
		t.Errorf("one catalog %q, want %q", got, want)
	}
}
