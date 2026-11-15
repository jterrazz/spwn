package apps

import (
	"slices"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

const EveryAppFollowsItsRule = "Every app follows its rule"

type Column struct {
	settings.LivesOn
	// value: when the machine answered; zero for a machine that was not read.
	Read time.Time
	// value: why a machine that should have answered did not, in the owner's words.
	Why string
}

type Cell struct {
	// value: the app's state on one machine; Not known where the machine was
	// not read.
	State State
	// value: the version it holds there, and the one an update would bring.
	Version string
	Newest  string
	// value: the app does not live on that machine: its list does not hold it.
	NotHere bool
	// value: the machine holds the app, installed in any form: the one
	// reading State.Installed gives, so a client counts holders itself.
	Holds bool
}

type MatrixRow struct {
	Name  string
	Cells []Cell
	// value: the machines that carry it, of those read that it lives on.
	Carry, Of int
}

type Matrix struct {
	Columns []Column
	Servers []string
	Rows    []MatrixRow
	// value: the head of the Apps shelf and of the Skills shelf.
	Verdict verdict.Shelf
	Skills  verdict.Shelf
}

type Reading struct {
	Reports []Report
	Read    bool
}

func MatrixOf(cols []Column, reads []Reading, servers []string, known []catalog.App) Matrix {
	// invariant: one row per app some machine carries, in the order the
	// machines list them, one cell per column; a machine not read says Not
	// known, one whose list lacks the app says it does not live there.
	m := Matrix{Columns: cols, Servers: servers}
	at := map[string]int{}
	kinds := map[string]catalog.AppKind{}
	for _, rd := range reads {
		for _, r := range rd.Reports {
			kinds[r.App.Name] = r.App.AppKind
			if _, ok := at[r.App.Name]; !ok {
				at[r.App.Name] = len(m.Rows)
				m.Rows = append(m.Rows, MatrixRow{Name: r.App.Name, Cells: slices.Repeat([]Cell{{State: NotKnown}}, len(cols))})
			}
		}
	}
	for i, rd := range reads {
		if !rd.Read {
			continue
		}
		listed := map[string]Report{}
		for _, r := range rd.Reports {
			listed[r.App.Name] = r
		}
		for j := range m.Rows {
			r, ok := listed[m.Rows[j].Name]
			if !ok {
				m.Rows[j].Cells[i] = Cell{State: NotInstalled, NotHere: true}
				continue
			}
			st := StateOf(r)
			m.Rows[j].Cells[i] = Cell{State: st, Version: r.Version, Newest: r.Newest, Holds: st.Installed()}
			m.Rows[j].Of++
			if st.Installed() {
				m.Rows[j].Carry++
			}
		}
	}
	m.Rows = slices.DeleteFunc(m.Rows, func(r MatrixRow) bool { return r.Carry == 0 })
	// invariant: a skill is counted on a shelf of its own: the catalog's
	// skills, and the skills some machine carries.
	skillsHeld := 0
	for _, r := range m.Rows {
		if kinds[r.Name] == catalog.AppKindSkill {
			skillsHeld++
		}
	}
	inCatalog := 0
	for _, a := range known {
		if a.AppKind == catalog.AppKindSkill {
			inCatalog++
		}
	}
	m.Verdict = verdict.Shelf{
		Mine:    verdict.Verdict{Tone: verdict.OK, Sentence: EveryAppFollowsItsRule},
		Catalog: verdict.Verdict{Tone: verdict.OK, Sentence: EveryAppFollowsItsRule},
	}
	m.Skills = verdict.Shelf{
		Mine:    verdict.Counted(skillsHeld, "skill", "on your machines"),
		Catalog: verdict.Counted(inCatalog, "skill", "in the catalog"),
	}
	return m
}
